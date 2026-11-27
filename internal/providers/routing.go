package providers

import (
	"context"
	"errors"
	"time"

	"calango/internal/core"
	"calango/internal/gateway"
	"calango/internal/harness"

	"github.com/IAPro-Community/IAPro-SDK/lease"
	"github.com/IAPro-Community/IAPro-SDK/model"
	"github.com/IAPro-Community/IAPro-SDK/routing"
)

var errRouteUnavailable = errors.New("capacity route unavailable: explicit intervention required")

const admissionTTL = time.Hour

// routeEvidence consumes only a current observation bound to the same login.
// Model-to-pool mapping is intentionally exact: never guess from a prefix.
func (s *Service) routeEvidence(ctx context.Context, owner string, ref core.ProviderRef) (routing.Account, error) {
	if s.capacity == nil || owner != s.cfg.CapacityOwner || ref.Kind != core.ProviderKindCodex || ref.Label == core.DefaultAccountLabel {
		return routing.Account{}, errRouteUnavailable
	}
	a, err := s.cfg.Repo.GetProviderAccountByName(ctx, owner, ref.Kind, ref.Label)
	if err != nil || a.ID != ref.AccountID {
		return routing.Account{}, errRouteUnavailable
	}
	m := s.capacity
	m.mu.Lock()
	e := m.entries[a.ID]
	if e == nil {
		m.mu.Unlock()
		return routing.Account{}, errRouteUnavailable
	}
	snap, adapter := e.snapshot, e.adapter
	m.mu.Unlock()
	current, err := adapter.Identity(owner, a.ID)
	if err != nil {
		return routing.Account{}, errRouteUnavailable
	}
	evidence, err := routing.VerifiedAccount(routing.EvidenceInput{
		OwnerID: owner, Scope: current.Scope, Snapshot: snap, Model: ref.Model, Now: s.cfg.Now(),
		Authorized: true, TermsPermit: s.cfg.RoutingAllowed, Authenticated: true, Health: string(model.HealthHealthy),
		Paid: !s.cfg.Prices.ConfirmedZero(ref.Key(), ref.Model, lease.PhysicalKey(owner, current.Scope).AccountID),
	})
	if err != nil {
		return routing.Account{}, errRouteUnavailable
	}
	return evidence, nil
}

// SelectCapacityRoute is a pre-dispatch decision only. A strict-mode turn
// never retries after calling the gateway. The configured chain authorizes
// candidates; the SDK is the authority for automatic fallback eligibility.
func (s *Service) SelectCapacityRoute(ctx context.Context, agent core.Agent, runID string) (core.ProviderRef, *gateway.CapacityRoute, error) {
	if !s.cfg.RoutingAllowed || s.leases == nil || agent.MaestroEnabled || len(agent.Providers) < 2 || runID == "" {
		return core.ProviderRef{}, nil, errRouteUnavailable
	}
	first := agent.Providers[0]
	source, err := s.routeEvidence(ctx, agent.OwnerID, first)
	if err != nil {
		return core.ProviderRef{}, nil, errRouteUnavailable
	}
	now := s.cfg.Now()
	makeRoute := func(ref core.ProviderRef, account routing.Account, reason string) (core.ProviderRef, *gateway.CapacityRoute, error) {
		return ref, &gateway.CapacityRoute{
			OwnerID: agent.OwnerID, RunID: runID, AccountKey: ref.Key(), Model: ref.Model,
			Scope: account.Scope, ExpiresAt: account.ExpiresAt, SourceKey: first.Key(), SourceScope: source.Scope, Reason: reason,
		}, nil
	}
	switch routing.AssessPrimary(source, now) {
	case routing.PrimaryAvailable:
		return makeRoute(first, source, "configured_primary")
	case routing.PrimaryExhausted:
		// Only verified exhaustion authorizes considering fallback.
	default:
		return core.ProviderRef{}, nil, errRouteUnavailable
	}

	candidates := make([]routing.Account, 0, len(agent.Providers)-1)
	refs := make(map[model.AccountScope]core.ProviderRef)
	seen := make(map[lease.Key]struct{})
	for _, ref := range agent.Providers[1:] {
		if ref.Model != first.Model {
			continue
		}
		candidate, err := s.routeEvidence(ctx, agent.OwnerID, ref)
		if err != nil {
			continue
		}
		key := lease.PhysicalKey(agent.OwnerID, candidate.Scope)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		refs[candidate.Scope] = ref
		candidates = append(candidates, candidate)
	}
	selected, err := routing.SelectFallback(routing.SwitchRequest{
		OwnerID: agent.OwnerID, RunID: runID, Source: source.Scope,
		AllowSwitch: true, AllowOtherProvider: false, AllowPaid: false,
		BeforeDispatch: true, NoOutput: true, NoSideEffects: true, Now: now,
	}, candidates)
	if err != nil {
		return core.ProviderRef{}, nil, errRouteUnavailable
	}
	return makeRoute(refs[selected.Scope], selected, "verified_capacity_fallback")
}

type leasedCodex struct {
	service *Service
	account core.ProviderAccount
	cli     *harness.Codex
}

var _ gateway.Provider = (*leasedCodex)(nil)
var _ gateway.AdmissionProvider = (*leasedCodex)(nil)

func (p *leasedCodex) Name() string { return p.cli.Name() }

func (p *leasedCodex) MayHaveSideEffects() bool { return true }

func (p *leasedCodex) Complete(ctx context.Context, req gateway.Request, stream func(gateway.Chunk)) (gateway.Usage, error) {
	usage, err := p.cli.Complete(ctx, req, stream)
	if p.service.leases == nil && req.CodexHome == "" {
		p.service.capacity.forget(p.account)
	}
	return usage, err
}

func (p *leasedCodex) Admit(ctx context.Context, req *gateway.Request) (func(error) error, error) {
	s := p.service
	if s.leases == nil {
		if req.CapacityRoute != nil {
			return nil, errRouteUnavailable
		}
		return nil, nil
	}
	if req.CodexHome != "" {
		if req.CapacityRoute != nil {
			return nil, errRouteUnavailable
		}
		// Maestro's separate runtime is outside this dedicated-profile policy.
		return nil, nil
	}
	adapter := harness.QuotaAdapter{Codex: p.cli, Now: s.cfg.Now}
	identity, err := adapter.Identity(p.account.OwnerID, p.account.ID)
	if err != nil {
		return nil, errRouteUnavailable
	}
	holder, err := s.cfg.NewID(s.cfg.Now())
	if err != nil || holder == "" {
		return nil, errRouteUnavailable
	}
	key := lease.PhysicalKey(p.account.OwnerID, identity.Scope)
	guard, err := s.leases.AcquireManaged(ctx, key, holder, lease.ManagedOptions{
		TTL: admissionTTL, RenewEvery: admissionTTL / 3, RetryEvery: 250 * time.Millisecond,
		StoreTimeout: 5 * time.Second, Wait: req.CapacityRoute == nil,
	})
	if err != nil {
		if errors.Is(err, lease.ErrReconcile) {
			s.capacity.markLeaseFailure(p.account, "lease_reconciliation_required")
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errRouteUnavailable
	}
	release := func(quiescent bool) error {
		if err := guard.Finish(quiescent); err != nil {
			reason := "lease_release_failed"
			if errors.Is(err, lease.ErrReconcile) {
				reason = "lease_reconciliation_required"
			}
			s.capacity.markLeaseFailure(p.account, reason)
			return errRouteUnavailable
		}
		return nil
	}
	fail := func() (func(error) error, error) {
		return nil, errors.Join(errRouteUnavailable, release(true))
	}
	check := func(checkCtx context.Context) error {
		valid, err := guard.Valid(checkCtx)
		if err != nil || !valid {
			return errRouteUnavailable
		}
		current, err := adapter.Identity(p.account.OwnerID, p.account.ID)
		if err != nil || current.Scope != identity.Scope {
			return errRouteUnavailable
		}
		if route := req.CapacityRoute; route != nil {
			ref := core.ProviderRef{AccountID: p.account.ID, Kind: p.account.Kind, Label: p.account.Label, Model: req.Model}
			evidence, err := s.routeEvidence(checkCtx, p.account.OwnerID, ref)
			if err != nil || route.OwnerID != p.account.OwnerID || route.AccountKey != req.Provider ||
				route.Model != req.Model || route.Scope != identity.Scope || evidence.Scope != route.Scope ||
				!s.cfg.Now().Before(route.ExpiresAt) {
				return errRouteUnavailable
			}
			switch route.Reason {
			case "configured_primary":
				if routing.AssessPrimary(evidence, s.cfg.Now()) != routing.PrimaryAvailable {
					return errRouteUnavailable
				}
			case "verified_capacity_fallback":
				_, err = routing.SelectFallback(routing.SwitchRequest{
					OwnerID: route.OwnerID, RunID: route.RunID, Source: route.SourceScope,
					AllowSwitch: true, BeforeDispatch: true, NoOutput: true, NoSideEffects: true, Now: s.cfg.Now(),
				}, []routing.Account{evidence})
				if err != nil {
					return errRouteUnavailable
				}
			default:
				return errRouteUnavailable
			}
		}
		return nil
	}
	if err := check(ctx); err != nil {
		return fail()
	}
	req.AdmissionScope = &identity.Scope
	req.AdmissionOwner = p.account.OwnerID
	req.CheckAdmission = check
	req.AdmissionContext = guard.Context()
	return func(result error) error {
		s.capacity.invalidateIdentity(identity.Scope)
		// The SDK owns renewal, fencing and release; Calango supplies its
		// accounting and dispatch evidence.
		return release(canReleaseAdmission(result))
	}, nil
}

// canReleaseAdmission admits only a completed turn or an explicit pre-model-
// dispatch refusal with durable accounting. A recorded error does not prove
// process-tree quiescence; cancellation, timeout and ambiguous provider
// failures retain the fence. Full process-tree containment is a separate gate.
func canReleaseAdmission(result error) bool {
	if errors.Is(result, gateway.ErrLedger) {
		return false
	}
	return result == nil || errors.Is(result, harness.ErrQuotaNotDispatched) ||
		errors.Is(result, harness.ErrNotInstalled) || errors.Is(result, harness.ErrSignedOut) ||
		errors.Is(result, harness.ErrQuotaIdentityChanged)
}
