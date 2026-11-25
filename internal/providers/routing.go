package providers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"calango/internal/core"
	"calango/internal/gateway"
	"calango/internal/harness"

	"github.com/IAPro-Community/IAPro-SDK/lease"
	"github.com/IAPro-Community/IAPro-SDK/model"
	"github.com/IAPro-Community/IAPro-SDK/quota"
	"github.com/IAPro-Community/IAPro-SDK/routing"
)

var errRouteUnavailable = errors.New("capacity route unavailable: explicit intervention required")

const admissionTTL = time.Hour

func accountLeaseKey(owner string, scope model.AccountScope) lease.Key {
	// Use the physical identity, not the profile or credential generation.
	// Do not persist the provider's raw account identifier.
	digest := sha256.Sum256([]byte(scope.AccountID))
	return lease.Key{OwnerID: owner, Provider: scope.ProviderID, AccountID: hex.EncodeToString(digest[:])}
}

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
	now := s.cfg.Now()
	current, err := adapter.Identity(owner, a.ID)
	if err != nil || current.Scope != snap.AccountScope || snap.ExpiresAt == nil ||
		!now.Before(*snap.ExpiresAt) || snap.FetchedAt.IsZero() || snap.FetchedAt.After(now) ||
		snap.Source != model.SourceOfficialAPI || snap.Confidence != model.UsageConfidenceHigh ||
		snap.ModelName == "" || snap.ModelName != ref.Model {
		return routing.Account{}, errRouteUnavailable
	}
	view := quota.BuildQuotaView(snap, "", "")
	view.ComputeAvailabilityAt(now)
	return routing.Account{
		OwnerID: owner, Scope: current.Scope, QuotaScope: snap.AccountScope,
		Model: ref.Model, QuotaGroup: "claude_gpt", GroupMapped: true,
		Authorized: true, TermsPermit: s.cfg.RoutingAllowed, ModelCompatible: true,
		Paid: !s.cfg.Prices.ConfirmedZeroTariff(ref.Key(), ref.Model), ExpiresAt: *snap.ExpiresAt,
		Eligibility: quota.EligibilityInput{View: view, Authenticated: true, Health: string(model.HealthHealthy)},
	}, nil
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
			Scope: account.Scope, ExpiresAt: account.ExpiresAt, SourceKey: first.Key(), Reason: reason,
		}, nil
	}
	status := source.Eligibility.View.Status
	if status == string(model.UsageLive) || status == string(model.UsageCached) {
		if quota.EvaluateEligibility(source.Eligibility).Eligible {
			return makeRoute(first, source, "configured_primary")
		}
		if source.Eligibility.View.AvailReasons.UnknownQuota || len(source.Eligibility.View.AvailReasons.ExhaustedWindows) == 0 {
			return core.ProviderRef{}, nil, errRouteUnavailable
		}
	} else if status != string(model.UsageRateLimited) {
		return core.ProviderRef{}, nil, errRouteUnavailable
	}

	candidates := make([]routing.Account, 0, len(agent.Providers)-1)
	refs := make(map[model.AccountScope]core.ProviderRef)
	for _, ref := range agent.Providers[1:] {
		if ref.Model != first.Model {
			continue
		}
		candidate, err := s.routeEvidence(ctx, agent.OwnerID, ref)
		if err != nil {
			continue
		}
		if _, exists := refs[candidate.Scope]; exists {
			continue
		}
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
	return p.cli.Complete(ctx, req, stream)
}

func (p *leasedCodex) Admit(ctx context.Context, req gateway.Request) (func(error) error, error) {
	s := p.service
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
	key := accountLeaseKey(p.account.OwnerID, identity.Scope)
	record, err := s.leases.Acquire(ctx, key, holder, admissionTTL, 0)
	if err != nil {
		return nil, errRouteUnavailable
	}
	release := func() error {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.leases.Release(cleanup, key, record, true)
	}
	fail := func() (func(error) error, error) {
		return nil, errors.Join(errRouteUnavailable, release())
	}
	if route := req.CapacityRoute; route != nil {
		ref := core.ProviderRef{AccountID: p.account.ID, Kind: p.account.Kind, Label: p.account.Label, Model: req.Model}
		evidence, err := s.routeEvidence(ctx, p.account.OwnerID, ref)
		if err != nil || route.OwnerID != p.account.OwnerID || route.AccountKey != req.Provider ||
			route.Model != req.Model || route.Scope != identity.Scope || evidence.Scope != route.Scope ||
			!s.cfg.Now().Before(route.ExpiresAt) || !quota.EvaluateEligibility(evidence.Eligibility).Eligible ||
			(route.Reason == "verified_capacity_fallback" && evidence.Paid) {
			return fail()
		}
	}
	valid, err := s.leases.Valid(ctx, key, record)
	if err != nil || !valid {
		return fail()
	}
	return func(result error) error {
		// Never release on unknown effects or an unsuccessful ledger write.
		// An expired record cannot be acquired without reconciliation.
		if result == nil || (!errors.Is(result, gateway.ErrLedger) &&
			(errors.Is(result, harness.ErrNotInstalled) || errors.Is(result, harness.ErrSignedOut) ||
				errors.Is(result, harness.ErrQuotaIdentityChanged))) {
			return release()
		}
		return nil
	}, nil
}
