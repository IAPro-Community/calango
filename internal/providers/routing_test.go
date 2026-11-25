package providers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IAPro-Community/IAPro-SDK/lease"
	"github.com/IAPro-Community/IAPro-SDK/model"

	"calango/internal/core"
	"calango/internal/gateway"
	"calango/internal/harness"
)

func TestAccountLeaseKeyUsesPhysicalIdentity(t *testing.T) {
	a := model.AccountScope{ProviderID: "codex", AccountID: "external", ProfileID: "profile-a", IdentityVersion: "v1"}
	b := a
	b.ProfileID, b.IdentityVersion = "profile-b", "v2"
	if accountLeaseKey("owner", a) != accountLeaseKey("owner", b) {
		t.Fatal("aliases or credential rotation created a second lease")
	}
	if accountLeaseKey("owner", a) == accountLeaseKey("other", a) {
		t.Fatal("owner isolation lost")
	}
}

func TestStrictRoutingDisabledFailsClosed(t *testing.T) {
	s := &Service{cfg: Config{Now: time.Now, Prices: gateway.PriceTable{}}}
	agent := core.Agent{OwnerID: "owner", CapacityRouting: true, Providers: []core.ProviderRef{{AccountID: "a"}, {AccountID: "b"}}}
	if _, route, err := s.SelectCapacityRoute(context.Background(), agent, "run"); err == nil || route != nil {
		t.Fatal("disabled capacity router admitted a run")
	}
}

func TestAdmissionReleaseRequiresSafeOutcome(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{nil, true},
		{harness.ErrNotInstalled, true},
		{harness.ErrSignedOut, true},
		{harness.ErrQuotaIdentityChanged, true},
		{errors.Join(harness.ErrQuotaNotDispatched, context.Canceled), true},
		{context.Canceled, false},
		{context.DeadlineExceeded, false},
		{gateway.ErrLedger, false},
		{errors.Join(gateway.ErrLedger, harness.ErrQuotaNotDispatched), false},
		{errors.New("ambiguous provider failure"), false},
	} {
		if got := canReleaseAdmission(tc.err); got != tc.want {
			t.Fatalf("release(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestAdmissionHeartbeatCancelsOnRenewalFailure(t *testing.T) {
	want := errors.New("lease unavailable")
	ctx, stop := startAdmissionHeartbeat(context.Background(), time.Millisecond, func(context.Context) error {
		return want
	})
	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("renewal failure did not cancel dispatch")
	}
	if err := stop(); !errors.Is(err, want) {
		t.Fatalf("stop = %v, want renewal failure", err)
	}
}

func TestLegacyAdmissionWaitsWithoutReclaimingExpiredFence(t *testing.T) {
	now := time.Now()
	coordinator, err := lease.New(&lease.MemoryStore{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	key := lease.Key{OwnerID: "owner", Provider: "codex", AccountID: "account"}
	first, err := coordinator.Acquire(context.Background(), key, "first", time.Hour, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acquireAccountLease(context.Background(), coordinator, key, "strict", false); !errors.Is(err, lease.ErrBusy) {
		t.Fatalf("strict admission = %v, want busy", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := acquireAccountLease(ctx, coordinator, key, "legacy", true); !errors.Is(err, context.Canceled) {
		t.Fatalf("legacy canceled admission = %v", err)
	}
	if err := coordinator.Release(context.Background(), key, first, true); err != nil {
		t.Fatal(err)
	}
	second, err := acquireAccountLease(context.Background(), coordinator, key, "legacy", true)
	if err != nil || second.Fence <= first.Fence {
		t.Fatalf("legacy acquisition = %+v, %v", second, err)
	}
	now = now.Add(2 * time.Hour)
	if _, err := acquireAccountLease(context.Background(), coordinator, key, "third", true); !errors.Is(err, lease.ErrReconcile) {
		t.Fatalf("expired acquisition = %v, want reconcile", err)
	}
}
