package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"calango/internal/core"
	"calango/internal/gateway"
	"calango/internal/usage"
)

func TestUsageTotalsScopesToOwnerAndCountsUnknownCost(t *testing.T) {
	ctx := context.Background()
	database := openTestStore(t)
	mine := strings.Repeat("a", 26)
	theirs := strings.Repeat("b", 26)
	for _, a := range []struct{ id, owner string }{{mine, "owner"}, {theirs, "other"}} {
		if _, err := database.db.ExecContext(ctx, `
			INSERT INTO agents(id, owner_id, name, token_limit_month, created_at, updated_at)
			VALUES (?, ?, 'Assistant', 1000, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
			a.id, a.owner); err != nil {
			t.Fatalf("insert agent: %v", err)
		}
	}

	known := int64(1000)
	jan := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	dec := time.Date(2026, 12, 5, 10, 0, 0, 0, time.UTC)
	records := []gateway.Record{
		{ID: strings.Repeat("1", 26), At: jan, Caller: gateway.Caller{AgentID: mine, RoutineID: "routine-1"},
			Provider: "fake", Model: "m", TokensIn: 10, TokensOut: 5, CostMicros: &known, CostStatus: gateway.CostKnown, PricingVersion: "v1"},
		{ID: strings.Repeat("2", 26), At: jan, Caller: gateway.Caller{AgentID: mine},
			Provider: "fake", Model: "m", TokensIn: 1, TokensOut: 1, CostStatus: gateway.CostUnknown},
		{ID: strings.Repeat("3", 26), At: dec, Caller: gateway.Caller{AgentID: mine, RoutineID: "routine-1"},
			Provider: "fake", Model: "m", TokensIn: 100, CostStatus: gateway.CostUnknown},
		{ID: strings.Repeat("4", 26), At: jan, Caller: gateway.Caller{AgentID: theirs},
			Provider: "fake", Model: "m", TokensIn: 1000, CostStatus: gateway.CostUnknown},
	}
	for _, rec := range records {
		if err := database.RecordUsage(ctx, rec); err != nil {
			t.Fatalf("RecordUsage(%s) error = %v", rec.ID, err)
		}
	}

	// To excludes the December row; the other owner's row never appears.
	byAgent, err := database.UsageTotals(ctx, "owner", usage.Query{
		GroupBy: usage.GroupAgent,
		To:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("UsageTotals(agent) error = %v", err)
	}
	want := []usage.Row{{Key: mine, TokensIn: 11, TokensOut: 6, CostMicros: 1000, UnknownCostRows: 1}}
	if len(byAgent) != 1 || byAgent[0] != want[0] {
		t.Fatalf("UsageTotals(agent, to=June) = %+v, want %+v", byAgent, want)
	}

	// Without bounds, a row without routine is grouped under the empty key, sorted first.
	byRoutine, err := database.UsageTotals(ctx, "owner", usage.Query{GroupBy: usage.GroupRoutine})
	if err != nil {
		t.Fatalf("UsageTotals(routine) error = %v", err)
	}
	wantRoutine := []usage.Row{
		{Key: "", TokensIn: 1, TokensOut: 1, CostMicros: 0, UnknownCostRows: 1},
		{Key: "routine-1", TokensIn: 110, TokensOut: 5, CostMicros: 1000, UnknownCostRows: 1},
	}
	if len(byRoutine) != len(wantRoutine) {
		t.Fatalf("UsageTotals(routine) = %+v, want %+v", byRoutine, wantRoutine)
	}
	for i := range wantRoutine {
		if byRoutine[i] != wantRoutine[i] {
			t.Fatalf("UsageTotals(routine)[%d] = %+v, want %+v", i, byRoutine[i], wantRoutine[i])
		}
	}

	if _, err := database.UsageTotals(ctx, "owner", usage.Query{GroupBy: usage.Group("bogus")}); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("UsageTotals(bogus group) error = %v, want ErrInvalidInput", err)
	}
}
