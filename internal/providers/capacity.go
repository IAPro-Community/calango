package providers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"calango/internal/core"
	"calango/internal/harness"

	capacity "github.com/IAPro-Community/IAPro-SDK/capacity"
	"github.com/IAPro-Community/IAPro-SDK/lease"
	"github.com/IAPro-Community/IAPro-SDK/model"
	sdk "github.com/IAPro-Community/IAPro-SDK/providers"
)

// CapacityView is intentionally not a UsageSnapshot: external account IDs,
// credential generations and untrusted diagnostic payloads never leave the host.
type CapacityView struct {
	AccountID  string           `json:"account_id"`
	Status     string           `json:"status"`
	Source     string           `json:"source"`
	Reason     string           `json:"reason"`
	ObservedAt *time.Time       `json:"observed_at,omitempty"`
	ExpiresAt  *time.Time       `json:"expires_at,omitempty"`
	Windows    []CapacityWindow `json:"windows"`
}

type CapacityWindow struct {
	Group     string     `json:"group"`
	Kind      string     `json:"kind"`
	State     string     `json:"state"`
	Remaining *float64   `json:"remaining_percent,omitempty"`
	ResetAt   *time.Time `json:"reset_at,omitempty"`
}

type capacityEntry struct {
	adapter  harness.QuotaAdapter
	cache    *sdk.UsageCache
	snapshot model.UsageSnapshot
	view     CapacityView
	next     time.Time
	previous map[capacity.ObservationKey]capacity.State
	failures int
}

type capacityMonitor struct {
	service *Service
	mu      sync.Mutex
	gate    chan struct{}
	entries map[string]*capacityEntry
}

const maxCapacityProfiles = 256

func newCapacityMonitor(s *Service) *capacityMonitor {
	return &capacityMonitor{service: s, gate: make(chan struct{}, 1), entries: make(map[string]*capacityEntry)}
}

func emptyCapacity(a core.ProviderAccount, status, reason string) CapacityView {
	return CapacityView{AccountID: a.ID, Status: status, Source: string(model.SourceNone), Reason: reason, Windows: []CapacityWindow{}}
}

func (s *Service) unsupportedCapacity(a core.ProviderAccount) (CapacityView, bool) {
	switch {
	case a.Kind != core.ProviderKindCodex:
		return emptyCapacity(a, string(model.UsageUnsupported), "provider_no_account_probe"), true
	case a.Label == core.DefaultAccountLabel:
		return emptyCapacity(a, string(model.UsageUnsupported), "shared_home"), true
	case s.capacity == nil || a.OwnerID != s.cfg.CapacityOwner:
		return emptyCapacity(a, string(model.UsageUnknown), "monitor_disabled"), true
	default:
		return CapacityView{}, false
	}
}

func (s *Service) capacityView(a core.ProviderAccount) CapacityView {
	if v, unsupported := s.unsupportedCapacity(a); unsupported {
		return v
	}
	m := s.capacity
	m.mu.Lock()
	e := m.entries[a.ID]
	if e == nil {
		m.mu.Unlock()
		return emptyCapacity(a, string(model.UsageUnknown), "not_observed")
	}
	view, snap, adapter := e.view, e.snapshot, e.adapter
	m.mu.Unlock()
	if snap.FetchedAt.IsZero() {
		return view
	}
	current, err := adapter.Identity(a.OwnerID, a.ID)
	if err != nil || current.Scope != snap.AccountScope {
		return emptyCapacity(a, string(model.UsageUnknown), "identity_changed")
	}
	now := s.cfg.Now()
	if view.ExpiresAt == nil || !now.Before(*view.ExpiresAt) {
		view.Status, view.Reason = "STALE", "observation_expired"
		view.Windows = []CapacityWindow{}
		return view
	}
	// Reproject individual windows at read time: their resets can precede TTL.
	return projectCapacity(a, snap, nil, now, s.cfg.CapacityInterval)
}

func projectCapacity(a core.ProviderAccount, snap model.UsageSnapshot, previous map[capacity.ObservationKey]capacity.State, now time.Time, ttl time.Duration) CapacityView {
	view := emptyCapacity(a, string(snap.Status), "")
	view.Source = string(snap.Source)
	view.ObservedAt = &snap.FetchedAt
	expires := snap.FetchedAt.Add(ttl)
	if snap.ExpiresAt != nil && snap.ExpiresAt.Before(expires) {
		expires = *snap.ExpiresAt
	}
	view.ExpiresAt = &expires
	if !now.Before(expires) {
		view.Status, view.Reason = "STALE", "observation_expired"
		return view
	}
	observations, err := capacity.Observe(a.OwnerID, snap, previous, capacity.DefaultThresholds(), now, ttl)
	if err != nil {
		view.Status, view.Reason = string(model.UsageError), "invalid_observation"
		return view
	}
	for _, o := range observations {
		window := CapacityWindow{Group: o.Key.Group, Kind: o.Key.Kind, State: string(o.State), Remaining: o.Remaining}
		for _, raw := range snap.Windows {
			if raw.Group == o.Key.Group && raw.Kind == o.Key.Kind {
				window.ResetAt = raw.ResetTime
				break
			}
		}
		view.Windows = append(view.Windows, window)
	}
	if snap.Status == model.UsageRateLimited {
		view.Reason = "rate_limited"
	}
	return view
}

func (m *capacityMonitor) invalidateIdentity(scope model.AccountScope) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, entry := range m.entries {
		seen := entry.snapshot.AccountScope
		if seen.ProviderID == scope.ProviderID && seen.AccountID == scope.AccountID {
			delete(m.entries, id)
		}
	}
}

// forget drops observations of an account the owner removed. Removing the
// account also removes quota evidence, so the change is announced instead of
// leaving the UI with a stale projection.
func (m *capacityMonitor) forget(a core.ProviderAccount) {
	m.mu.Lock()
	_, existed := m.entries[a.ID]
	delete(m.entries, a.ID)
	m.mu.Unlock()
	if !existed || m.service.cfg.OnCapacity == nil {
		return
	}
	view := emptyCapacity(a, string(model.UsageUnsupported), "account_forgotten")
	m.service.cfg.OnCapacity(a.OwnerID, view)
}

// markLeaseFailure removes quota evidence and makes an unsafe reservation
// visible to the owner. Only an actual successful probe can restore evidence.
func (m *capacityMonitor) markLeaseFailure(a core.ProviderAccount, reason string) {
	view := emptyCapacity(a, string(model.UsageError), reason)
	m.mu.Lock()
	e := m.entries[a.ID]
	if e == nil {
		// Keep an error-only entry until the next refresh can rebuild its
		// adapter. Bound this cache exactly like regular observations.
		if len(m.entries) < maxCapacityProfiles {
			m.entries[a.ID] = &capacityEntry{view: view}
		}
		m.mu.Unlock()
		if m.service.cfg.OnCapacity != nil {
			m.service.cfg.OnCapacity(a.OwnerID, view)
		}
		return
	}
	changed := capacityChanged(e.view, view)
	e.view, e.snapshot, e.previous = view, model.UsageSnapshot{}, nil
	e.next = m.service.cfg.Now().Add(m.service.cfg.CapacityInterval)
	m.mu.Unlock()
	if changed && m.service.cfg.OnCapacity != nil {
		m.service.cfg.OnCapacity(a.OwnerID, view)
	}
}

// RefreshCapacity is bounded by one daemon-wide maintenance job. A concurrent
// request returns the current projection rather than spawning another probe.
func (s *Service) RefreshCapacity(ctx context.Context, owner, id string) (View, error) {
	accounts, err := s.cfg.Repo.ListProviderAccounts(ctx, owner)
	if err != nil {
		return View{}, err
	}
	for _, a := range accounts {
		if a.ID == id {
			if _, unsupported := s.unsupportedCapacity(a); !unsupported {
				s.capacity.refresh(ctx, a)
			}
			return s.view(a), ctx.Err()
		}
	}
	return View{}, fmt.Errorf("%w: provider account", core.ErrNotFound)
}

func (m *capacityMonitor) refresh(ctx context.Context, a core.ProviderAccount) {
	select {
	case m.gate <- struct{}{}:
		defer func() { <-m.gate }()
	default:
		return
	}
	s := m.service
	parentCtx := ctx
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	currentAccount, err := s.cfg.Repo.GetProviderAccountByName(ctx, a.OwnerID, a.Kind, a.Label)
	if err != nil || currentAccount.ID != a.ID {
		return
	}
	now := s.cfg.Now()
	m.mu.Lock()
	e := m.entries[a.ID]
	if e == nil || e.cache == nil {
		if e == nil && len(m.entries) >= maxCapacityProfiles {
			m.mu.Unlock()
			return
		}
		cli, err := harness.NewCodex(s.cfg.CodexBin, s.cfg.CodexWorkDir, s.codexHome(a))
		if err != nil {
			m.mu.Unlock()
			return
		}
		adapter := harness.QuotaAdapter{Codex: cli, Now: s.cfg.Now}
		cache, err := sdk.NewUsageCache(adapter, adapter, s.cfg.CapacityInterval, s.cfg.CapacityInterval, 4, s.cfg.Now)
		if err != nil {
			m.mu.Unlock()
			return
		}
		if e == nil {
			e = &capacityEntry{view: emptyCapacity(a, string(model.UsageUnknown), "not_observed")}
		}
		e.adapter, e.cache = adapter, cache
		m.entries[a.ID] = e
	}
	if now.Before(e.next) {
		m.mu.Unlock()
		return
	}
	oldView := e.view
	previous := e.previous
	m.mu.Unlock()
	account, err := e.adapter.Identity(a.OwnerID, a.ID)
	var snap model.UsageSnapshot
	if err == nil {
		var reservation lease.Record
		var key lease.Key
		if s.leases != nil {
			holder, idErr := s.cfg.NewID(s.cfg.Now())
			if idErr != nil {
				return
			}
			key = lease.PhysicalKey(a.OwnerID, account.Scope)
			reservation, err = s.leases.Acquire(ctx, key, holder, admissionTTL, 0)
			if err != nil {
				if errors.Is(err, lease.ErrReconcile) {
					m.markLeaseFailure(a, "lease_reconciliation_required")
				} else if !errors.Is(err, lease.ErrBusy) && ctx.Err() == nil {
					m.markLeaseFailure(a, "lease_unavailable")
				}
				return
			}
			// Keep the fence until the observation has been published. Otherwise
			// a completed run could invalidate before this pre-run snapshot is stored.
			defer func() {
				cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
				defer stop()
				if s.leases.Release(cleanup, key, reservation, true) != nil {
					m.markLeaseFailure(a, "lease_release_failed")
				}
			}()
		}
		snap, err = e.cache.GetUsage(ctx, account)
	}
	if parentCtx.Err() != nil {
		return
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	now = s.cfg.Now()
	view := emptyCapacity(a, string(model.UsageError), "probe_failed")
	states := make(map[capacity.ObservationKey]capacity.State)
	if err == nil {
		view = projectCapacity(a, snap, previous, now, s.cfg.CapacityInterval)
		observations, _ := capacity.Observe(a.OwnerID, snap, previous, capacity.DefaultThresholds(), now, s.cfg.CapacityInterval)
		for _, o := range observations {
			states[o.Key] = o.State
		}
	} else if account.Scope == (model.AccountScope{}) {
		view = emptyCapacity(a, string(model.UsageUnknown), "identity_unverified")
	}
	m.mu.Lock()
	if m.entries[a.ID] != e {
		m.mu.Unlock()
		return
	}
	e.view, e.snapshot, e.previous = view, snap, states
	e.next = now.Add(s.cfg.CapacityInterval)
	if err != nil || view.Status == string(model.UsageError) {
		e.failures++
		if e.failures > 3 {
			e.failures = 3
		}
		e.next = now.Add(s.cfg.CapacityInterval * time.Duration(1<<e.failures))
	} else {
		e.failures = 0
	}
	failures := e.failures
	m.mu.Unlock()
	persistentFailure := failures == 2
	if s.cfg.OnCapacity != nil && (persistentFailure || (capacityChanged(oldView, view) && failures == 0)) {
		s.cfg.OnCapacity(a.OwnerID, view)
	}
}

func capacityChanged(a, b CapacityView) bool {
	if a.Status != b.Status || a.Reason != b.Reason || len(a.Windows) != len(b.Windows) {
		return true
	}
	for i, w := range a.Windows {
		next := b.Windows[i]
		if w.Group != next.Group || w.Kind != next.Kind || w.State != next.State {
			return true
		}
	}
	return false
}

// RunCapacity performs provider I/O only in this worker or an explicit refresh;
// listProviderAccounts is a read model. No cached observation survives restart.
func (s *Service) RunCapacity(ctx context.Context) {
	if s.capacity == nil {
		return
	}
	ticker := time.NewTicker(s.cfg.CapacityInterval)
	defer ticker.Stop()
	for {
		accounts, err := s.cfg.Repo.ListProviderAccounts(ctx, s.cfg.CapacityOwner)
		if err == nil {
			for _, a := range accounts {
				if ctx.Err() != nil {
					return
				}
				if _, unsupported := s.unsupportedCapacity(a); !unsupported {
					s.capacity.refresh(ctx, a)
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
