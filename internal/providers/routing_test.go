package providers

import (
	"context"
	"errors"
	"testing"
	"time"

	"calango/internal/core"
	"calango/internal/gateway"
	"calango/internal/harness"
)

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

