package providers

import (
	"context"
	"errors"
	"testing"

	"calango/internal/gateway"
	"calango/internal/harness"
)

func TestAdmissionReleaseRequiresSafeOutcome(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"success", nil, true},
		{"ambiguous provider failure recorded", errors.New("ambiguous provider failure"), false},
		{"context canceled after dispatch", context.Canceled, false},
		{"deadline exceeded after dispatch", context.DeadlineExceeded, false},
		{"not installed", harness.ErrNotInstalled, true},
		{"signed out", harness.ErrSignedOut, true},
		{"identity changed", harness.ErrQuotaIdentityChanged, true},
		{"not dispatched", errors.Join(harness.ErrQuotaNotDispatched, context.Canceled), true},
		{"model not supported after dispatch", harness.ErrModelNotSupported, false},
		{"model not supported before dispatch", errors.Join(harness.ErrModelNotSupported, harness.ErrQuotaNotDispatched), true},
		{"ledger failure", gateway.ErrLedger, false},
		{"ledger plus not dispatched", errors.Join(gateway.ErrLedger, harness.ErrQuotaNotDispatched), false},
	} {
		if got := canReleaseAdmission(tc.err); got != tc.want {
			t.Fatalf("%s: release(%v) = %v, want %v", tc.name, tc.err, got, tc.want)
		}
	}
}
