package providers

import (
	"errors"
	"testing"
	"time"

	"github.com/IAPro-Community/IAPro-SDK/model"
	sdk "github.com/IAPro-Community/IAPro-SDK/providers"

	"calango/internal/core"
)

func TestCapacityProjectionDistinguishesZeroUnknownAndExpired(t *testing.T) {
	now := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	a := core.ProviderAccount{ID: "profile", OwnerID: "owner", Kind: "codex"}
	zero := 0.0
	reset := now.Add(time.Minute)
	scope := model.AccountScope{ProviderID: "codex", ProfileID: a.ID, AccountID: "external", IdentityVersion: "generation"}
	snap := model.UsageSnapshot{
		ProviderID: "codex", ProfileID: a.ID, AccountScope: scope,
		Status: model.UsageLive, Source: model.SourceOfficialAPI, FetchedAt: now,
		Windows: []model.UsageWindow{
			{Kind: "5h", Group: "pool", RemainingPercent: &zero, ResetTime: &reset},
			{Kind: "weekly", Group: "pool"},
		},
	}
	view := projectCapacity(a, snap, nil, now, 5*time.Minute)
	if len(view.Windows) != 2 || view.Windows[0].Remaining == nil || *view.Windows[0].Remaining != 0 ||
		view.Windows[0].State != "EXHAUSTED" || view.Windows[1].Remaining != nil || view.Windows[1].State != "UNKNOWN" {
		t.Fatalf("unexpected projection: %+v", view)
	}
	view = projectCapacity(a, snap, nil, reset, 5*time.Minute)
	if view.Windows[0].Remaining != nil || view.Windows[0].State != "UNKNOWN" {
		t.Fatalf("expired window: %+v", view.Windows[0])
	}
	view = projectCapacity(a, snap, nil, now.Add(5*time.Minute), 5*time.Minute)
	if view.Status != "STALE" || len(view.Windows) != 0 {
		t.Fatalf("expired observation: %+v", view)
	}
}

func TestCapacityUnsupportedAccounts(t *testing.T) {
	s := &Service{}
	for _, a := range []core.ProviderAccount{
		{ID: "api", Kind: core.ProviderKindAnthropic},
		{ID: "shared", Kind: core.ProviderKindCodex, Label: core.DefaultAccountLabel},
	} {
		view := s.capacityView(a)
		if view.Status != string(model.UsageUnsupported) || len(view.Windows) != 0 {
			t.Fatalf("unsupported account: %+v", view)
		}
	}
	a := core.ProviderAccount{ID: "dedicated", Kind: core.ProviderKindCodex, Label: "work"}
	if view := s.capacityView(a); view.Status != string(model.UsageUnknown) || view.Reason != "monitor_disabled" {
		t.Fatalf("disabled monitor: %+v", view)
	}
}

func TestLeaseFailureInvalidatesQuotaAndPublishesOnce(t *testing.T) {
	now := time.Now()
	a := core.ProviderAccount{ID: "profile", OwnerID: "owner", Kind: core.ProviderKindCodex, Label: "work"}
	var events []CapacityView
	s := &Service{cfg: Config{
		Now: time.Now, CapacityOwner: a.OwnerID, CapacityInterval: time.Minute,
		OnCapacity: func(owner string, view CapacityView) {
			if owner != a.OwnerID {
				t.Fatal("wrong event owner")
			}
			events = append(events, view)
		},
	}}
	s.capacity = newCapacityMonitor(s)
	s.capacity.entries[a.ID] = &capacityEntry{
		snapshot: model.UsageSnapshot{FetchedAt: now},
		view:     CapacityView{Status: string(model.UsageLive)},
	}
	s.capacity.markLeaseFailure(a, "lease_reconciliation_required")
	s.capacity.markLeaseFailure(a, "lease_reconciliation_required")
	view := s.capacityView(a)
	if view.Status != string(model.UsageError) || view.Reason != "lease_reconciliation_required" || len(view.Windows) != 0 || len(events) != 1 {
		t.Fatalf("view=%+v events=%d", view, len(events))
	}
}

func TestForgetPublishesRemovalAndStaysQuietWhenAbsent(t *testing.T) {
	a := core.ProviderAccount{ID: "profile", OwnerID: "owner", Kind: core.ProviderKindCodex, Label: "work"}
	var events []CapacityView
	s := &Service{cfg: Config{
		Now:              time.Now,
		CapacityOwner:    a.OwnerID,
		CapacityInterval: time.Minute,
		OnCapacity: func(owner string, view CapacityView) {
			if owner != a.OwnerID {
				t.Fatal("wrong event owner")
			}
			events = append(events, view)
		},
	}}
	s.capacity = newCapacityMonitor(s)
	// Nothing cached: forgetting must not announce a phantom change.
	s.capacity.forget(a)
	if len(events) != 0 {
		t.Fatalf("absent account published %d events", len(events))
	}
	s.capacity.entries[a.ID] = &capacityEntry{view: CapacityView{Status: string(model.UsageLive)}}
	s.capacity.forget(a)
	if _, ok := s.capacity.entries[a.ID]; ok {
		t.Fatal("entry survived forget")
	}
	if len(events) != 1 || events[0].Status != string(model.UsageUnsupported) || events[0].Reason != "account_forgotten" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestCapacityProbeFailureUsesSDKAuthDiagnostic(t *testing.T) {
	a := core.ProviderAccount{ID: "profile", OwnerID: "owner", Kind: core.ProviderKindCodex}
	scope := model.AccountScope{ProviderID: "codex", ProfileID: a.ID, AccountID: "external", IdentityVersion: "generation"}
	for _, tc := range []struct {
		scope  model.AccountScope
		err    error
		status model.UsageStatus
		reason string
	}{
		{scope, &sdk.ProbeError{Code: sdk.ProbeAuthExpired, Message: "private"}, model.UsageError, "auth_required"},
		{scope, errors.New("unknown"), model.UsageError, "probe_failed"},
		{model.AccountScope{}, sdk.ErrIdentityMismatch, model.UsageUnknown, "identity_unverified"},
	} {
		view := capacityProbeFailure(a, tc.scope, tc.err)
		if view.Status != string(tc.status) || view.Reason != tc.reason || len(view.Windows) != 0 {
			t.Fatalf("unexpected failure projection: %+v", view)
		}
	}
}
