package memory_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"calango/internal/agents"
	"calango/internal/core"
	"calango/internal/memory"
	"calango/internal/store"
)

const ownerA = "owner-a"

// clock advances one millisecond per reading, so memories created later sort later.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(time.Millisecond)
	return c.t
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type fixture struct {
	t      *testing.T
	clk    *clock
	svc    *memory.Service
	agents *agents.Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	st, err := store.OpenMemory(context.Background())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	clk := &clock{t: time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)}
	svc, err := memory.NewService(st, clk.now, core.NewID)
	if err != nil {
		t.Fatalf("memory service: %v", err)
	}
	ag, err := agents.NewService(st, clk.now, core.NewID)
	if err != nil {
		t.Fatalf("agents service: %v", err)
	}
	return &fixture{t: t, clk: clk, svc: svc, agents: ag}
}

func (f *fixture) agent(owner, name string) string {
	f.t.Helper()
	view, err := f.agents.Create(context.Background(), owner, core.Agent{
		Name: name, Model: "fake-model", Harness: "fake", TokenLimitMonth: 1000,
	})
	if err != nil {
		f.t.Fatalf("create agent %s: %v", name, err)
	}
	return view.ID
}

func (f *fixture) create(owner string, m core.Memory) core.Memory {
	f.t.Helper()
	got, err := f.svc.Create(context.Background(), owner, m)
	if err != nil {
		f.t.Fatalf("create memory %q: %v", m.Text, err)
	}
	return got
}

func accountMemory(text string) core.Memory {
	return core.Memory{Scope: core.ScopeAccount, Kind: core.KindFact, Text: text}
}

func agentMemory(agentID, text string) core.Memory {
	return core.Memory{Scope: core.ScopeAgent, AgentID: agentID, Kind: core.KindFact, Text: text}
}

func TestCreateValidatesAndRejectsForeignAgent(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ownAgent := f.agent(ownerA, "Calango")
	foreignAgent := f.agent("owner-b", "Outro")

	if _, err := f.svc.Create(ctx, ownerA, core.Memory{Scope: core.ScopeAccount, Text: "   "}); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("blank text error = %v, want ErrInvalidInput", err)
	}
	if _, err := f.svc.Create(ctx, ownerA, core.Memory{Scope: core.ScopeAgent, Text: "x"}); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("agent scope without agent error = %v, want ErrInvalidInput", err)
	}
	if _, err := f.svc.Create(ctx, ownerA, agentMemory(foreignAgent, "x")); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("foreign agent error = %v, want ErrNotFound", err)
	}
	if _, err := f.svc.Create(ctx, ownerA, agentMemory(ownAgent, strings.Repeat("a", core.MaxMemoryChars+1))); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("oversized text error = %v, want ErrInvalidInput", err)
	}

	got := f.create(ownerA, core.Memory{Scope: core.ScopeAccount, Text: "  Prefere respostas curtas  "})
	if got.Text != "Prefere respostas curtas" || got.Kind != core.KindFact || got.OwnerID != ownerA {
		t.Fatalf("created memory = %+v, want trimmed text, default kind and owner", got)
	}
}

func TestContextIncludesOnlyVisibleActiveMemories(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	calango := f.agent(ownerA, "Calango")
	other := f.agent(ownerA, "Outro agente")

	f.create(ownerA, accountMemory("Conta visível"))
	f.create(ownerA, agentMemory(calango, "Agente visível"))
	f.create(ownerA, agentMemory(other, "Outro agente oculto"))
	deleted := f.create(ownerA, accountMemory("Apagada"))
	f.create("owner-b", accountMemory("Outro dono oculto"))
	if err := f.svc.Delete(ctx, ownerA, deleted.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	block, err := f.svc.Context(ctx, ownerA, calango)
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	for _, want := range []string{"Conta visível", "Agente visível"} {
		if !strings.Contains(block, want) {
			t.Fatalf("context misses %q:\n%s", want, block)
		}
	}
	for _, unwanted := range []string{"Outro agente oculto", "Apagada", "Outro dono oculto"} {
		if strings.Contains(block, unwanted) {
			t.Fatalf("context leaks %q:\n%s", unwanted, block)
		}
	}
}

func TestContextIsEmptyWithoutMemories(t *testing.T) {
	f := newFixture(t)
	block, err := f.svc.Context(context.Background(), ownerA, f.agent(ownerA, "Calango"))
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	if block != "" {
		t.Fatalf("context = %q, want empty so prompts are unchanged", block)
	}
}

func TestContextRespectsBudgetAndKeepsMostRecent(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	calango := f.agent(ownerA, "Calango")

	var last core.Memory
	for i := 0; i < 20; i++ {
		last = f.create(ownerA, accountMemory(fmt.Sprintf("%02d-%s", i, strings.Repeat("x", 400))))
	}

	block, err := f.svc.Context(ctx, ownerA, calango)
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	if tokens := core.EstimateTokens(block); tokens > memory.ContextBudget {
		t.Fatalf("context uses %d tokens, budget is %d", tokens, memory.ContextBudget)
	}
	if !strings.Contains(block, last.Text) {
		t.Fatalf("context dropped the most recent memory")
	}
	if strings.Contains(block, "00-"+strings.Repeat("x", 400)) {
		t.Fatalf("context kept the oldest memory although the budget is full")
	}
}

func TestUpdateAndDeleteTakeEffectAtOnce(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	m := f.create(ownerA, accountMemory("Versão um"))

	updated, err := f.svc.Update(ctx, ownerA, m.ID, "  Versão dois  ")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Text != "Versão dois" {
		t.Fatalf("updated text = %q", updated.Text)
	}

	if err := f.svc.Delete(ctx, ownerA, m.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := f.svc.Get(ctx, ownerA, m.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("get after delete error = %v, want ErrNotFound", err)
	}
	if _, err := f.svc.Update(ctx, ownerA, m.ID, "depois"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("update after delete error = %v, want ErrNotFound", err)
	}
	if err := f.svc.Delete(ctx, ownerA, m.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("second delete error = %v, want ErrNotFound", err)
	}
	list, err := f.svc.List(ctx, ownerA, "", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("list after delete = %+v, want empty", list)
	}
}

func TestOtherOwnerCannotReadOrChangeMemory(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	m := f.create(ownerA, accountMemory("Privado"))

	if _, err := f.svc.Get(ctx, "owner-b", m.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("foreign get error = %v, want ErrNotFound", err)
	}
	if _, err := f.svc.Update(ctx, "owner-b", m.ID, "roubado"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("foreign update error = %v, want ErrNotFound", err)
	}
	if err := f.svc.Delete(ctx, "owner-b", m.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("foreign delete error = %v, want ErrNotFound", err)
	}
	got, err := f.svc.Get(ctx, ownerA, m.ID)
	if err != nil || got.Text != "Privado" {
		t.Fatalf("owner memory after foreign attempts = %+v, %v", got, err)
	}
}

func TestPurgeRemovesOnlyDeletesPastRetention(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	old := f.create(ownerA, accountMemory("Apagada há muito"))
	if err := f.svc.Delete(ctx, ownerA, old.ID); err != nil {
		t.Fatalf("delete old: %v", err)
	}

	f.clk.advance(31 * 24 * time.Hour)
	recent := f.create(ownerA, accountMemory("Apagada agora"))
	if err := f.svc.Delete(ctx, ownerA, recent.ID); err != nil {
		t.Fatalf("delete recent: %v", err)
	}

	n, err := f.svc.Purge(ctx)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if n != 1 {
		t.Fatalf("first purge removed %d rows, want 1 (only the memory past retention)", n)
	}

	f.clk.advance(31 * 24 * time.Hour)
	n, err = f.svc.Purge(ctx)
	if err != nil {
		t.Fatalf("second purge: %v", err)
	}
	if n != 1 {
		t.Fatalf("second purge removed %d rows, want 1", n)
	}
}
