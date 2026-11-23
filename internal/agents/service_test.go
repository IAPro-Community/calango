package agents_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"calango/internal/agents"
	"calango/internal/core"
	"calango/internal/gateway"
	"calango/internal/store"
)

var fixedNow = time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

func newService(t *testing.T) (*agents.Service, *store.Store) {
	t.Helper()
	database, err := store.OpenMemory(context.Background())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	svc, err := agents.NewService(database, func() time.Time { return fixedNow }, core.NewID)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc, database
}

func validInput() core.Agent {
	return core.Agent{Name: "Assistant", Instructions: "Be brief", TokenLimitMonth: 1000}
}

func TestCreateRefusesAgentWithoutUsableConfiguration(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)

	noLimit := validInput()
	noLimit.TokenLimitMonth = 0
	blankName := validInput()
	blankName.Name = "   "

	for name, in := range map[string]core.Agent{"no limit": noLimit, "blank name": blankName} {
		if _, err := svc.Create(ctx, "owner", in); !errors.Is(err, core.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if list, err := svc.List(ctx, "owner"); err != nil || len(list) != 0 {
		t.Fatalf("refused agents were stored: list=%d err=%v", len(list), err)
	}
}

func TestCreateStoresEveryFieldAndDefaults(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)

	created, err := svc.Create(ctx, "owner", core.Agent{
		Name: "  Researcher  ", Role: "reads papers", Instructions: "cite sources",
		Harness: "claude-code", Model: "m-1", ReasoningEffort: core.EffortHigh,
		RunOn: core.RunOnVPS, TokenLimitMonth: 5000, ApprovalLevel: core.ApprovalTrusted,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := svc.Get(ctx, "owner", created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Researcher" || got.Role != "reads papers" || got.Instructions != "cite sources" ||
		got.Harness != "claude-code" || got.Model != "m-1" || got.ReasoningEffort != core.EffortHigh ||
		got.RunOn != core.RunOnVPS || got.TokenLimitMonth != 5000 || got.ApprovalLevel != core.ApprovalTrusted {
		t.Fatalf("stored configuration differs from input: %+v", got.Agent)
	}
	if !got.CreatedAt.Equal(fixedNow) || len(got.ID) != 26 {
		t.Fatalf("created_at=%v id=%q", got.CreatedAt, got.ID)
	}

	defaults, err := svc.Create(ctx, "owner", validInput())
	if err != nil {
		t.Fatalf("create with defaults: %v", err)
	}
	if defaults.RunOn != core.RunOnPC || defaults.ApprovalLevel != core.ApprovalSensitive || defaults.ReasoningEffort != "" {
		t.Fatalf("defaults = run_on %q, approval %q, effort %q", defaults.RunOn, defaults.ApprovalLevel, defaults.ReasoningEffort)
	}
	if defaults.Paused || defaults.TokensUsedMonth != 0 {
		t.Fatalf("new agent should be unpaused with no usage: %+v", defaults)
	}
}

func TestReplaceAppliesNewConfigurationAndKeepsIdentity(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)

	created, err := svc.Create(ctx, "owner", validInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	updated := validInput()
	updated.Instructions = "answer in one line"
	updated.TokenLimitMonth = 2000

	view, err := svc.Replace(ctx, "owner", created.ID, updated)
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if view.ID != created.ID || !view.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("replace changed identity: id=%q created=%v", view.ID, view.CreatedAt)
	}
	got, err := svc.Get(ctx, "owner", created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Instructions != "answer in one line" || got.TokenLimitMonth != 2000 {
		t.Fatalf("stored agent = %+v", got.Agent)
	}

	invalid := validInput()
	invalid.TokenLimitMonth = 0
	if _, err := svc.Replace(ctx, "owner", created.ID, invalid); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("replace with no limit: error = %v, want ErrInvalidInput", err)
	}
	after, _ := svc.Get(ctx, "owner", created.ID)
	if after.TokenLimitMonth != 2000 {
		t.Fatalf("refused replace changed the limit to %d", after.TokenLimitMonth)
	}
}

func TestAgentsAreVisibleOnlyToTheirOwner(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)

	created, err := svc.Create(ctx, "owner-a", validInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Get(ctx, "owner-b", created.ID); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("get by other owner: error = %v, want ErrNotFound", err)
	}
	if list, _ := svc.List(ctx, "owner-b"); len(list) != 0 {
		t.Errorf("other owner lists %d agents", len(list))
	}
	if err := svc.Delete(ctx, "owner-b", created.ID); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("delete by other owner: error = %v, want ErrNotFound", err)
	}
}

func TestDeleteRefusedWhenAgentHasUsageHistory(t *testing.T) {
	ctx := context.Background()
	svc, database := newService(t)

	created, err := svc.Create(ctx, "owner", validInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := database.RecordUsage(ctx, gateway.Record{
		ID: strings.Repeat("9", 26), At: fixedNow,
		Caller:   gateway.Caller{AgentID: created.ID},
		Provider: "fake", Model: "m", TokensIn: 1, TokensOut: 1, CostStatus: gateway.CostUnknown,
	}); err != nil {
		t.Fatalf("record usage: %v", err)
	}

	if err := svc.Delete(ctx, "owner", created.ID); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("delete with history: error = %v, want ErrConflict", err)
	}
	if _, err := svc.Get(ctx, "owner", created.ID); err != nil {
		t.Fatalf("agent with history disappeared: %v", err)
	}
}

func TestDeleteRemovesAgentWithoutHistory(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)

	created, err := svc.Create(ctx, "owner", validInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Delete(ctx, "owner", created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get(ctx, "owner", created.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("get after delete: error = %v, want ErrNotFound", err)
	}
}

func TestPauseAllCoversNewAgentsUntilResumed(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)

	first, err := svc.Create(ctx, "owner", validInput())
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if err := svc.PauseAll(ctx); err != nil {
		t.Fatalf("pause all: %v", err)
	}
	later, err := svc.Create(ctx, "owner", validInput())
	if err != nil {
		t.Fatalf("create while paused: %v", err)
	}
	for _, id := range []string{first.ID, later.ID} {
		v, err := svc.Get(ctx, "owner", id)
		if err != nil || !v.Paused {
			t.Fatalf("agent %s paused=%v err=%v, want paused", id, v.Paused, err)
		}
	}

	if err := svc.ResumeAll(ctx); err != nil {
		t.Fatalf("resume all: %v", err)
	}
	list, err := svc.List(ctx, "owner")
	if err != nil || len(list) != 2 {
		t.Fatalf("list = %d, err=%v", len(list), err)
	}
	for _, v := range list {
		if v.Paused {
			t.Fatalf("agent %s still paused after resume", v.ID)
		}
	}
}

func TestMonthlyUsageCountsOnlyTheCurrentUTCMonth(t *testing.T) {
	ctx := context.Background()
	svc, database := newService(t)

	created, err := svc.Create(ctx, "owner", validInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	usage := []struct {
		id string
		at time.Time
		n  int64
	}{
		{strings.Repeat("1", 26), time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC), 500},
		{strings.Repeat("2", 26), time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), 40},
		{strings.Repeat("3", 26), time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC), 2},
	}
	for _, u := range usage {
		if err := database.RecordUsage(ctx, gateway.Record{
			ID: u.id, At: u.at, Caller: gateway.Caller{AgentID: created.ID},
			Provider: "fake", Model: "m", TokensIn: u.n, CostStatus: gateway.CostUnknown,
		}); err != nil {
			t.Fatalf("record usage at %v: %v", u.at, err)
		}
	}

	view, err := svc.Get(ctx, "owner", created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if view.TokensUsedMonth != 42 {
		t.Fatalf("TokensUsedMonth = %d, want 42 (October excluded, 1 Nov inclusive)", view.TokensUsedMonth)
	}
}
