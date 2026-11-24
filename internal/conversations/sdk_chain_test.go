package conversations

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"calango/internal/gateway"
	"calango/internal/harness"
)

func TestSDKChainSafety(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		err     error
		emitted bool
		effects bool
		want    bool
	}{
		{"signed out before dispatch", harness.ErrSignedOut, false, true, true},
		{"missing CLI before dispatch", harness.ErrNotInstalled, false, true, true},
		{"direct text unavailable", gateway.ErrUnavailable, false, false, true},
		{"ambiguous tool execution", gateway.ErrUnavailable, false, true, false},
		{"output already sent", gateway.ErrUnavailable, true, false, false},
		{"budget", gateway.ErrBudgetExceeded, false, false, false},
		{"ledger", gateway.ErrLedger, false, false, false},
		{"wrapped cancellation", fmt.Errorf("%w: %w", gateway.ErrUnavailable, context.Canceled), false, false, false},
		{"unknown failure", errors.New("unknown"), false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := canAdvanceChain(ctx, tc.err, tc.emitted, tc.effects); got != tc.want {
				t.Fatalf("advance = %v, want %v", got, tc.want)
			}
		})
	}
}
