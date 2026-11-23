package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"calango/internal/gateway"
)

func TestUsageLedgerSumsTokensPerAgentAndMonth(t *testing.T) {
	ctx := context.Background()
	database := openTestStore(t)
	agentID := strings.Repeat("a", 26)
	if _, err := database.db.ExecContext(ctx, `
		INSERT INTO agents(id, owner_id, name, token_limit_month, created_at, updated_at)
		VALUES (?, 'owner', 'Assistant', 1000, '2026-11-01T00:00:00Z', '2026-11-01T00:00:00Z')`, agentID); err != nil {
		t.Fatalf("insert agent: %v", err)
	}

	limit, err := database.AgentTokenLimit(ctx, agentID)
	if err != nil || limit != 1000 {
		t.Fatalf("AgentTokenLimit() = (%d, %v), want (1000, nil)", limit, err)
	}
	if _, err := database.AgentTokenLimit(ctx, strings.Repeat("z", 26)); !errors.Is(err, gateway.ErrUnknownAgent) {
		t.Fatalf("AgentTokenLimit(unknown) error = %v, want ErrUnknownAgent", err)
	}

	known := int64(10_500_000)
	records := []gateway.Record{
		{ID: strings.Repeat("1", 26), At: time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC), Caller: gateway.Caller{AgentID: agentID, RoutineID: "routine-1"},
			Provider: "fake", Model: "m", TokensIn: 100, TokensOut: 50, CostMicros: &known, CostStatus: gateway.CostKnown, PricingVersion: "v1"},
		{ID: strings.Repeat("2", 26), At: time.Date(2026, 11, 30, 23, 59, 59, 500_000_000, time.UTC), Caller: gateway.Caller{AgentID: agentID},
			Provider: "fake", Model: "m", TokensIn: 1, TokensOut: 2, CostStatus: gateway.CostUnknown},
		{ID: strings.Repeat("3", 26), At: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), Caller: gateway.Caller{AgentID: agentID},
			Provider: "fake", Model: "m", TokensIn: 1000, TokensOut: 1000, CostStatus: gateway.CostUnknown},
	}
	for _, rec := range records {
		if err := database.RecordUsage(ctx, rec); err != nil {
			t.Fatalf("RecordUsage(%s) error = %v", rec.ID, err)
		}
	}

	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	got, err := database.MonthlyTokens(ctx, agentID, from, to)
	if err != nil {
		t.Fatalf("MonthlyTokens() error = %v", err)
	}
	if got != 100+50+1+2 {
		t.Fatalf("MonthlyTokens(November) = %d, want 153 (December row excluded, fractional-second row included)", got)
	}
}

func TestUsageLedgerStoresUnknownCostAsNull(t *testing.T) {
	ctx := context.Background()
	database := openTestStore(t)
	agentID := strings.Repeat("a", 26)
	if _, err := database.db.ExecContext(ctx, `
		INSERT INTO agents(id, owner_id, name, token_limit_month, created_at, updated_at)
		VALUES (?, 'owner', 'Assistant', 1000, '2026-11-01T00:00:00Z', '2026-11-01T00:00:00Z')`, agentID); err != nil {
		t.Fatalf("insert agent: %v", err)
	}

	rec := gateway.Record{ID: strings.Repeat("1", 26), At: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC),
		Caller: gateway.Caller{AgentID: agentID}, Provider: "fake", Model: "unpriced", TokensIn: 1, CostStatus: gateway.CostUnknown}
	if err := database.RecordUsage(ctx, rec); err != nil {
		t.Fatalf("RecordUsage() error = %v", err)
	}

	var cost any
	var status string
	if err := database.db.QueryRowContext(ctx,
		`SELECT cost_micros, cost_status FROM usage WHERE id = ?`, rec.ID).Scan(&cost, &status); err != nil {
		t.Fatalf("read usage row: %v", err)
	}
	if cost != nil || status != "unknown" {
		t.Fatalf("stored (cost_micros, cost_status) = (%v, %q), want (NULL, unknown)", cost, status)
	}
}
