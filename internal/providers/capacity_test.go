package providers

import (
	"testing"
	"time"

	"github.com/IAPro-Community/IAPro-SDK/model"

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
