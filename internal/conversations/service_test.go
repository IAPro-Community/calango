package conversations_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"calango/internal/agents"
	"calango/internal/conversations"
	"calango/internal/core"
	"calango/internal/gateway"
	"calango/internal/memory"
	"calango/internal/store"
)

const owner = "owner-a"

// fakeProvider answers with fixed chunks. When release is set, it blocks until
// Release is called or the call's context is cancelled.
type fakeProvider struct {
	chunks  []string
	head    []string
	release chan struct{}
	once    sync.Once

	// failure is returned before any chunk streams; tail after them.
	failure error
	tail    error
	effects bool

	mu                 sync.Mutex
	codexHomes         []string
	prompts            []string
	activeCalls        int
	maxConcurrentCalls int
}

func (p *fakeProvider) Name() string { return "fake" }

func (p *fakeProvider) MayHaveSideEffects() bool { return p.effects }

func (p *fakeProvider) Release() {
	if p.release != nil {
		p.once.Do(func() { close(p.release) })
	}
}

func (p *fakeProvider) Prompts() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.prompts...)
}

func (p *fakeProvider) CodexHomes() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.codexHomes...)
}

func (p *fakeProvider) MaxConcurrentCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.maxConcurrentCalls
}

func (p *fakeProvider) Complete(ctx context.Context, req gateway.Request, stream func(gateway.Chunk)) (gateway.Usage, error) {
	p.mu.Lock()
	p.prompts = append(p.prompts, req.Prompt)
	p.activeCalls++
	if p.activeCalls > p.maxConcurrentCalls {
		p.maxConcurrentCalls = p.activeCalls
	}
	p.codexHomes = append(p.codexHomes, req.CodexHome)
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.activeCalls--
		p.mu.Unlock()
	}()

	for _, c := range p.head {
		if stream != nil {
			stream(gateway.Chunk{Text: c})
		}
	}

	if p.release != nil {
		select {
		case <-p.release:
		case <-ctx.Done():
			return gateway.Usage{}, ctx.Err()
		}
	}
	if p.failure != nil {
		return gateway.Usage{}, p.failure
	}
	for _, c := range p.chunks {
		if stream != nil {
			stream(gateway.Chunk{Text: c})
		}
	}
	if p.tail != nil {
		return gateway.Usage{}, p.tail
	}
	return gateway.Usage{TokensIn: 10, TokensOut: 5}, nil
}

// clock advances one millisecond per reading, so ids made later sort later.
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

type env struct {
	t        *testing.T
	st       *store.Store
	clk      *clock
	gw       *gateway.Gateway
	svc      *conversations.Service
	agents   *agents.Service
	memory   *memory.Service
	provider *fakeProvider
	agentID  string
	conv     core.Conversation
}

func newEnv(t *testing.T, fp *fakeProvider, tokenLimit int64, parallel int) *env {
	t.Helper()
	e := newEnvWith(t, providerSet{"fake/default": fp}, []chainEntry{{label: "default", model: "fake-model"}}, tokenLimit, parallel)
	e.provider = fp
	return e
}

// chainEntry is one account in an agent's provider chain; an empty kind defaults to fake.
type chainEntry struct {
	kind  string
	label string
	model string
}

// providerSet maps account keys ("kind/label") to fake providers, the way the
// daemon's resolver maps keys to real accounts. Unknown keys fail as unconfigured.
type providerSet map[string]*fakeProvider

func (ps providerSet) resolve(_ context.Context, name string) (gateway.Provider, error) {
	p, ok := ps[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", gateway.ErrUnknownProvider, name)
	}
	return p, nil
}

func newEnvWith(t *testing.T, set providerSet, chain []chainEntry, tokenLimit int64, parallel int) *env {
	t.Helper()
	return newEnvWithUsageID(t, set, chain, tokenLimit, parallel, core.NewID)
}

func newEnvWithUsageID(t *testing.T, set providerSet, chain []chainEntry, tokenLimit int64, parallel int, usageID func(time.Time) (string, error)) *env {
	t.Helper()
	ctx := context.Background()

	st, err := store.OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	// Cleanups run last-in first-out: the store closes after the service has
	// finished, and the providers are released before the service is waited on.
	t.Cleanup(func() { _ = st.Close() })

	clk := &clock{t: time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)}
	gw, err := gateway.New(gateway.Config{
		Ledger: st, Resolve: set.resolve, Now: clk.now, NewID: usageID,
	})
	if err != nil {
		t.Fatalf("gateway: %v", err)
	}
	ag, err := agents.NewService(st, clk.now, core.NewID)
	if err != nil {
		t.Fatalf("agents service: %v", err)
	}
	mem, err := memory.NewService(st, clk.now, core.NewID)
	if err != nil {
		t.Fatalf("memory service: %v", err)
	}
	e := &env{t: t, st: st, clk: clk, gw: gw, agents: ag, memory: mem}
	e.svc = e.newService()
	t.Cleanup(e.svc.Wait)
	for _, fp := range set {
		t.Cleanup(fp.Release)
	}

	refs := make([]core.ProviderRef, 0, len(chain))
	for _, c := range chain {
		id, err := core.NewID(clk.now())
		if err != nil {
			t.Fatalf("account id: %v", err)
		}
		now := clk.now()
		kind := c.kind
		if kind == "" {
			kind = "fake"
		}
		acc := core.ProviderAccount{ID: id, OwnerID: owner, Kind: kind, Label: c.label, CreatedAt: now, UpdatedAt: now}
		if err := st.InsertProviderAccount(ctx, acc); err != nil {
			t.Fatalf("insert provider account: %v", err)
		}
		refs = append(refs, core.ProviderRef{AccountID: id, Kind: kind, Label: c.label, Model: c.model})
	}

	view, err := ag.Create(ctx, owner, core.Agent{
		Name: "Calango", Providers: refs, TokenLimitMonth: tokenLimit,
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	e.agentID = view.ID
	e.conv, err = e.svc.CreateDM(ctx, owner, view.ID, "")
	if err != nil {
		t.Fatalf("create dm: %v", err)
	}
	e.parallelLimit(parallel)
	return e
}

func (e *env) parallelLimit(n int) {
	e.t.Helper()
	if n == 0 {
		return
	}
	// The service under test is rebuilt with the requested limit; the agent
	// and conversation created above live in the same store.
	e.svc = e.newServiceWith(n)
	e.t.Cleanup(e.svc.Wait)
}

func (e *env) newService() *conversations.Service {
	return e.newServiceWith(0)
}

func (e *env) newServiceWith(parallel int) *conversations.Service {
	e.t.Helper()
	svc, err := conversations.NewService(conversations.Config{
		Repo: e.st, Gateway: e.gw, Memory: e.memory,
		ParallelLimit: parallel, Now: e.clk.now, NewID: core.NewID,
	})
	if err != nil {
		e.t.Fatalf("conversations service: %v", err)
	}
	return svc
}

func waitStatus(t *testing.T, e *env, runID string, want core.RunStatus) core.Run {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		detail, err := e.svc.GetRun(context.Background(), owner, runID)
		if err != nil {
			t.Fatalf("get run %s: %v", runID, err)
		}
		if detail.Run.Status == want {
			return detail.Run
		}
		if time.Now().After(deadline) {
			t.Fatalf("run %s status = %s, want %s", runID, detail.Run.Status, want)
		}
		time.Sleep(time.Millisecond)
	}
}

func waitPromptCount(t *testing.T, fp *fakeProvider, want int) []string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		prompts := fp.Prompts()
		if len(prompts) >= want {
			return prompts
		}
		if time.Now().After(deadline) {
			t.Fatalf("provider received %d prompts, want at least %d", len(prompts), want)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestMessageRunsToDoneAndRecordsSteps(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"olá, ", "mundo"}}
	e := newEnv(t, fp, 1000000, 0)

	msg, run, err := e.svc.SendMessage(context.Background(), owner, e.conv.ID, "oi", "key-1")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if run.Status != core.RunRunning {
		t.Fatalf("admitted run status = %s, want running (free slot)", run.Status)
	}
	if msg.Body != "oi" || run.SourceMessageID != msg.ID {
		t.Fatalf("run %+v does not point at message %+v", run, msg)
	}

	done := waitStatus(t, e, run.ID, core.RunDone)
	if done.TokensIn != 10 || done.TokensOut != 5 {
		t.Fatalf("tokens = %d/%d, want 10/5", done.TokensIn, done.TokensOut)
	}

	detail, err := e.svc.GetRun(context.Background(), owner, run.ID)
	if err != nil {
		t.Fatalf("get run detail: %v", err)
	}
	if len(detail.Steps) != 3 || detail.Steps[0].Seq != 0 || detail.Steps[1].Seq != 1 || detail.Steps[2].Kind != core.StepProviderAttempt {
		t.Fatalf("steps = %+v, want two text steps then one provider attempt", detail.Steps)
	}
	if detail.Steps[0].Payload != `{"text":"olá, "}` {
		t.Fatalf("first step payload = %s", detail.Steps[0].Payload)
	}
	if detail.Steps[2].Payload != `{"account":"fake/default","error_code":"","model":"fake-model","outcome":"done"}` {
		t.Fatalf("attempt payload = %s", detail.Steps[2].Payload)
	}
	if prompts := fp.Prompts(); len(prompts) != 1 || prompts[0] != "oi" {
		t.Fatalf("prompts = %v, want [oi]", prompts)
	}
}

func TestLinkedAgentUsesResolvedMaestroCodexHome(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"ok"}}
	e := newEnvWith(t, providerSet{"codex/default": fp}, []chainEntry{
		{kind: "codex", label: "default", model: "gpt-test"},
	}, 1000000, 0)
	ctx := context.Background()

	agent, err := e.agents.Get(ctx, owner, e.agentID)
	if err != nil {
		t.Fatal(err)
	}
	agent.MaestroEnabled = true
	if _, err := e.agents.Replace(ctx, owner, e.agentID, agent.Agent); err != nil {
		t.Fatal(err)
	}

	home := filepath.Join(t.TempDir(), "maestro", ".codex")
	service, err := conversations.NewService(conversations.Config{
		Repo: e.st, Gateway: e.gw, Memory: e.memory,
		MaestroHome: func() (string, error) { return home, nil },
		Now:         e.clk.now, NewID: core.NewID,
	})
	if err != nil {
		t.Fatal(err)
	}
	e.svc = service
	t.Cleanup(service.Wait)

	_, run, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "oi", "maestro-home")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	waitStatus(t, e, run.ID, core.RunDone)
	homes := fp.CodexHomes()
	if len(homes) != 1 || homes[0] != home {
		t.Fatalf("Codex homes = %q, want only %q", homes, home)
	}
}

func TestLinkedAgentFailsClosedWhenMaestroHomeIsUnavailable(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"should not run"}}
	e := newEnvWith(t, providerSet{"codex/default": fp}, []chainEntry{
		{kind: "codex", label: "default", model: "gpt-test"},
	}, 1000000, 0)
	ctx := context.Background()

	agent, err := e.agents.Get(ctx, owner, e.agentID)
	if err != nil {
		t.Fatal(err)
	}
	agent.MaestroEnabled = true
	if _, err := e.agents.Replace(ctx, owner, e.agentID, agent.Agent); err != nil {
		t.Fatal(err)
	}

	service, err := conversations.NewService(conversations.Config{
		Repo: e.st, Gateway: e.gw, Memory: e.memory,
		MaestroHome: func() (string, error) { return "", errors.New("home unavailable") },
		Now:         e.clk.now, NewID: core.NewID,
	})
	if err != nil {
		t.Fatal(err)
	}
	e.svc = service
	t.Cleanup(service.Wait)

	_, run, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "oi", "maestro-unavailable")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	failed := waitStatus(t, e, run.ID, core.RunFailed)
	if failed.ErrorCode != core.ErrCodeMaestroUnavailable {
		t.Fatalf("run error code = %q, want %q", failed.ErrorCode, core.ErrCodeMaestroUnavailable)
	}
	if prompts := fp.Prompts(); len(prompts) != 0 {
		t.Fatalf("provider was called despite unavailable Maestro home: %q", prompts)
	}
}

func TestProviderFailureDoesNotLeakIntoRunStepsOrLogs(t *testing.T) {
	const canary = "provider-error-secret-canary-84a2"
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	e := newEnv(t, &fakeProvider{failure: errors.New(canary)}, 1000000, 0)
	_, run, err := e.svc.SendMessage(context.Background(), owner, e.conv.ID, "oi", "secret-error")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	waitStatus(t, e, run.ID, core.RunFailed)
	detail, err := e.svc.GetRun(context.Background(), owner, run.ID)
	if err != nil {
		t.Fatalf("get failed run: %v", err)
	}
	for _, step := range detail.Steps {
		if strings.Contains(step.Payload, canary) {
			t.Fatalf("run step contains provider error secret: %s", step.Payload)
		}
	}
	if strings.Contains(logs.String(), canary) {
		t.Fatalf("application log contains provider error secret: %s", logs.String())
	}
}

func TestRunPromptCarriesOnlyActiveMemories(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"ok"}}
	e := newEnv(t, fp, 1000000, 0)
	ctx := context.Background()

	accountMem, err := e.memory.Create(ctx, owner, core.Memory{
		Scope: core.ScopeAccount, Kind: core.KindPreference, Text: "Responde em português",
	})
	if err != nil {
		t.Fatalf("create account memory: %v", err)
	}
	if _, err := e.memory.Create(ctx, owner, core.Memory{
		Scope: core.ScopeAgent, AgentID: e.agentID, Kind: core.KindFact, Text: "Trabalha com Go",
	}); err != nil {
		t.Fatalf("create agent memory: %v", err)
	}

	_, run1, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "oi", "key-1")
	if err != nil {
		t.Fatalf("send first message: %v", err)
	}
	waitStatus(t, e, run1.ID, core.RunDone)

	if err := e.memory.Delete(ctx, owner, accountMem.ID); err != nil {
		t.Fatalf("delete account memory: %v", err)
	}
	_, run2, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "de novo", "key-2")
	if err != nil {
		t.Fatalf("send second message: %v", err)
	}
	waitStatus(t, e, run2.ID, core.RunDone)

	prompts := fp.Prompts()
	if len(prompts) != 2 {
		t.Fatalf("prompts = %d, want 2", len(prompts))
	}
	if !strings.Contains(prompts[0], "Responde em português") || !strings.Contains(prompts[0], "Trabalha com Go") {
		t.Fatalf("first prompt misses active memories: %q", prompts[0])
	}
	if !strings.HasSuffix(prompts[0], "\n\noi") {
		t.Fatalf("first prompt must end with the message: %q", prompts[0])
	}
	if strings.Contains(prompts[1], "Responde em português") {
		t.Fatalf("deleted memory reached the next run: %q", prompts[1])
	}
	if !strings.Contains(prompts[1], "Trabalha com Go") {
		t.Fatalf("agent memory missing from the next run: %q", prompts[1])
	}
	if !strings.HasSuffix(prompts[1], "\n\nde novo") {
		t.Fatalf("second prompt must end with the message: %q", prompts[1])
	}
}

func TestRepeatedKeyReturnsSameMessageAndRun(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"ok"}}
	e := newEnv(t, fp, 1000000, 0)
	ctx := context.Background()

	msg1, run1, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "oi", "same-key")
	if err != nil {
		t.Fatalf("first send: %v", err)
	}
	msg2, run2, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "oi", "same-key")
	if err != nil {
		t.Fatalf("repeated send: %v", err)
	}
	if msg1.ID != msg2.ID || run1.ID != run2.ID {
		t.Fatalf("repeated send created new resources: message %s/%s run %s/%s",
			msg1.ID, msg2.ID, run1.ID, run2.ID)
	}
	waitStatus(t, e, run1.ID, core.RunDone)

	messages, err := e.svc.ListMessages(ctx, owner, e.conv.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("stored messages = %d, want the person message and agent reply", len(messages))
	}
	if messages[0].ID != msg1.ID {
		t.Fatalf("first message = %+v, want the submitted message %s", messages[0], msg1.ID)
	}
	reply := messages[1]
	if reply.AuthorKind != core.AuthorAgent || reply.AuthorID != e.agentID || reply.Body != "ok" {
		t.Fatalf("agent reply = %+v, want agent %s with body %q", reply, e.agentID, "ok")
	}
	if prompts := fp.Prompts(); len(prompts) != 1 {
		t.Fatalf("model called %d times, want 1", len(prompts))
	}
}

func TestStopEndsOnlyThatRunAndQueuedRunStarts(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"x"}, release: make(chan struct{})}
	e := newEnv(t, fp, 1000000, 2)
	ctx := context.Background()

	_, a, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "a", "key-a")
	if err != nil {
		t.Fatalf("send a: %v", err)
	}
	_, b, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "b", "key-b")
	if err != nil {
		t.Fatalf("send b: %v", err)
	}
	_, c, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "c", "key-c")
	if err != nil {
		t.Fatalf("send c: %v", err)
	}
	if a.Status != core.RunRunning {
		t.Fatalf("a = %s, want running", a.Status)
	}
	if b.Status != core.RunQueued || c.Status != core.RunQueued {
		t.Fatalf("b=%s c=%s, want both queued behind the active agent run", b.Status, c.Status)
	}

	if err := e.svc.Stop(ctx, owner, a.ID); err != nil {
		t.Fatalf("stop a: %v", err)
	}
	waitStatus(t, e, a.ID, core.RunStopped)
	waitStatus(t, e, b.ID, core.RunRunning)
	waitStatus(t, e, c.ID, core.RunQueued)

	fp.Release()
	waitStatus(t, e, b.ID, core.RunDone)
	waitStatus(t, e, c.ID, core.RunDone)
}

func TestPausedNodeRecordsRunWithoutCallingModel(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"x"}}
	e := newEnv(t, fp, 1000000, 0)
	ctx := context.Background()

	if err := e.agents.PauseAll(ctx); err != nil {
		t.Fatalf("pause all: %v", err)
	}
	msg, run, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "oi", "paused-key")
	if err != nil {
		t.Fatalf("send while paused: %v", err)
	}
	if run.Status != core.RunFailed || run.ErrorCode != core.ErrCodeAgentPaused {
		t.Fatalf("run = %s/%q, want failed/agent_paused", run.Status, run.ErrorCode)
	}
	if run.EndedAt == nil {
		t.Fatalf("paused run has no end time")
	}
	if prompts := fp.Prompts(); len(prompts) != 0 {
		t.Fatalf("model was called while paused: %v", prompts)
	}
	if messages, _ := e.svc.ListMessages(ctx, owner, e.conv.ID); len(messages) != 1 || messages[0].ID != msg.ID {
		t.Fatalf("message not kept while paused: %+v", messages)
	}
}

func TestStartInterruptsRunsLeftByCrash(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"x"}, release: make(chan struct{})}
	e := newEnv(t, fp, 1000000, 0)
	ctx := context.Background()

	_, run, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "oi", "crash-key")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if run.Status != core.RunRunning {
		t.Fatalf("run = %s, want running before the simulated crash", run.Status)
	}

	// A second service on the same store stands in for the restarted process.
	// The first service's goroutine is still blocked; it is released in cleanup.
	restarted := e.newService()
	if err := restarted.Start(ctx); err != nil {
		t.Fatalf("start after crash: %v", err)
	}
	detail, err := restarted.GetRun(ctx, owner, run.ID)
	if err != nil {
		t.Fatalf("get run after restart: %v", err)
	}
	if detail.Run.Status != core.RunFailed || detail.Run.ErrorCode != core.ErrCodeInterrupted {
		t.Fatalf("run after restart = %s/%q, want failed/interrupted",
			detail.Run.Status, detail.Run.ErrorCode)
	}
}

func TestRetryRedoesStoppedRunWithInstructions(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"x"}, release: make(chan struct{})}
	e := newEnv(t, fp, 1000000, 0)
	ctx := context.Background()

	msg, a, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "oi", "first-key")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(fp.Prompts()) < 1 {
		if time.Now().After(deadline) {
			t.Fatal("first run never reached the provider")
		}
		time.Sleep(time.Millisecond)
	}
	if err := e.svc.Stop(ctx, owner, a.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	waitStatus(t, e, a.ID, core.RunStopped)

	r, err := e.svc.Retry(ctx, owner, a.ID, "use bullet points", "retry-key")
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if r.RetryOf != a.ID || r.Instructions != "use bullet points" {
		t.Fatalf("retry run = %+v, want retry_of %s with instructions", r, a.ID)
	}
	if r.SourceMessageID != msg.ID || r.ConversationID != e.conv.ID {
		t.Fatalf("retry run %+v is not in the same place as the original", r)
	}

	if _, err := e.svc.Retry(ctx, owner, r.ID, "again", "retry-key-2"); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("retry of a running run error = %v, want ErrConflict", err)
	}

	fp.Release()
	waitStatus(t, e, r.ID, core.RunDone)
	prompts := fp.Prompts()
	if len(prompts) != 2 || !strings.Contains(prompts[1], "Instructions for this retry:\nuse bullet points") {
		t.Fatalf("prompts = %v, want retry prompt to carry the instructions", prompts)
	}
}

func TestBudgetExceededFailsRunBeforeProvider(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"x"}}
	e := newEnv(t, fp, 1, 0)
	ctx := context.Background()

	_, run, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "oi", "budget-key")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	failed := waitStatus(t, e, run.ID, core.RunFailed)
	if failed.ErrorCode != core.ErrCodeBudgetExceeded {
		t.Fatalf("error code = %q, want budget_exceeded", failed.ErrorCode)
	}
	if prompts := fp.Prompts(); len(prompts) != 0 {
		t.Fatalf("provider called despite exhausted budget: %v", prompts)
	}
}

func TestOtherOwnerCannotReachConversation(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"x"}}
	e := newEnv(t, fp, 1000000, 0)
	ctx := context.Background()

	if _, _, err := e.svc.SendMessage(ctx, "intruder", e.conv.ID, "oi", "intruder-key"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("send as intruder error = %v, want ErrNotFound", err)
	}
	if _, err := e.svc.ListMessages(ctx, "intruder", e.conv.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("list as intruder error = %v, want ErrNotFound", err)
	}
}

func TestWaitRunHoldsWhileRunWaitsForApproval(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"x"}, release: make(chan struct{})}
	e := newEnv(t, fp, 1000000, 0)
	ctx := context.Background()

	_, r, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "hi", "key-wait")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	waitStatus(t, e, r.ID, core.RunRunning)
	if moved, err := e.st.SetRunStatus(ctx, r.ID, core.RunRunning, core.RunWaitingApproval); err != nil || !moved {
		t.Fatalf("to waiting_approval: moved=%v err=%v", moved, err)
	}

	done := make(chan core.Run, 1)
	failed := make(chan error, 1)
	go func() {
		detail, err := e.svc.WaitRun(ctx, owner, r.ID)
		if err != nil {
			failed <- err
			return
		}
		done <- detail.Run
	}()
	select {
	case run := <-done:
		t.Fatalf("WaitRun returned while the run waits for approval: status %s", run.Status)
	case err := <-failed:
		t.Fatalf("WaitRun: %v", err)
	case <-time.After(300 * time.Millisecond):
	}

	if moved, err := e.st.SetRunStatus(ctx, r.ID, core.RunWaitingApproval, core.RunRunning); err != nil || !moved {
		t.Fatalf("back to running: moved=%v err=%v", moved, err)
	}
	fp.Release()
	select {
	case run := <-done:
		if run.Status != core.RunDone {
			t.Fatalf("WaitRun returned status %s, want done", run.Status)
		}
	case err := <-failed:
		t.Fatalf("WaitRun: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("WaitRun did not return after the run finished")
	}
}

func TestAgentRunsAreSerializedAcrossDMs(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"reply"}, release: make(chan struct{})}
	e := newEnv(t, fp, 1000000, 3)
	defer fp.Release()
	ctx := context.Background()
	secondConversation, err := e.svc.CreateDM(ctx, owner, e.agentID, "")
	if err != nil {
		t.Fatalf("create second dm: %v", err)
	}

	_, first, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "first request", "key-first-dm")
	if err != nil {
		t.Fatalf("send first: %v", err)
	}
	if first.Status != core.RunRunning {
		t.Fatalf("first run = %s, want running", first.Status)
	}
	waitPromptCount(t, fp, 1)

	_, queued, err := e.svc.SendMessage(ctx, owner, secondConversation.ID, "second request", "key-second-dm")
	if err != nil {
		t.Fatalf("send second: %v", err)
	}
	if queued.Status != core.RunQueued {
		t.Fatalf("second run = %s, want queued while the agent's first run is active", queued.Status)
	}
	if got := len(fp.Prompts()); got != 1 {
		t.Fatalf("provider received %d prompts before first completed, want 1", got)
	}

	fp.Release()
	waitStatus(t, e, first.ID, core.RunDone)
	waitStatus(t, e, queued.ID, core.RunDone)
	if got := fp.MaxConcurrentCalls(); got != 1 {
		t.Fatalf("maximum concurrent provider calls = %d, want 1", got)
	}
}

func TestWaitingApprovalHoldsAgentSlotAcrossDMs(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"reply"}, release: make(chan struct{})}
	e := newEnv(t, fp, 1000000, 3)
	defer fp.Release()
	ctx := context.Background()
	secondConversation, err := e.svc.CreateDM(ctx, owner, e.agentID, "")
	if err != nil {
		t.Fatalf("create second dm: %v", err)
	}

	_, first, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "approval request", "key-approval-dm")
	if err != nil {
		t.Fatalf("send first: %v", err)
	}
	waitPromptCount(t, fp, 1)
	if moved, err := e.st.SetRunStatus(ctx, first.ID, core.RunRunning, core.RunWaitingApproval); err != nil || !moved {
		t.Fatalf("to waiting_approval: moved=%v err=%v", moved, err)
	}

	_, queued, err := e.svc.SendMessage(ctx, owner, secondConversation.ID, "second request", "key-after-approval")
	if err != nil {
		t.Fatalf("send second: %v", err)
	}
	if queued.Status != core.RunQueued {
		t.Fatalf("second run = %s, want queued while first waits for approval", queued.Status)
	}
	if moved, err := e.st.SetRunStatus(ctx, first.ID, core.RunWaitingApproval, core.RunRunning); err != nil || !moved {
		t.Fatalf("back to running: moved=%v err=%v", moved, err)
	}
	if run := waitStatus(t, e, queued.ID, core.RunQueued); run.Status != core.RunQueued {
		t.Fatalf("second run = %s after approval wait, want queued until first completes", run.Status)
	}

	fp.Release()
	waitStatus(t, e, first.ID, core.RunDone)
	waitStatus(t, e, queued.ID, core.RunDone)
	if got := fp.MaxConcurrentCalls(); got != 1 {
		t.Fatalf("maximum concurrent provider calls = %d, want 1", got)
	}
}

func TestCancelingQueuedAgentRunDoesNotStarveNext(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"reply"}, release: make(chan struct{})}
	e := newEnv(t, fp, 1000000, 3)
	defer fp.Release()
	ctx := context.Background()
	secondConversation, err := e.svc.CreateDM(ctx, owner, e.agentID, "")
	if err != nil {
		t.Fatalf("create second dm: %v", err)
	}
	thirdConversation, err := e.svc.CreateDM(ctx, owner, e.agentID, "")
	if err != nil {
		t.Fatalf("create third dm: %v", err)
	}

	_, first, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "first request", "key-cancel-first")
	if err != nil {
		t.Fatalf("send first: %v", err)
	}
	waitPromptCount(t, fp, 1)
	_, canceled, err := e.svc.SendMessage(ctx, owner, secondConversation.ID, "canceled request", "key-cancel-second")
	if err != nil {
		t.Fatalf("send second: %v", err)
	}
	_, next, err := e.svc.SendMessage(ctx, owner, thirdConversation.ID, "next request", "key-cancel-third")
	if err != nil {
		t.Fatalf("send third: %v", err)
	}
	if canceled.Status != core.RunQueued || next.Status != core.RunQueued {
		t.Fatalf("queued runs before cancellation: canceled=%s next=%s", canceled.Status, next.Status)
	}
	if err := e.svc.Stop(ctx, owner, canceled.ID); err != nil {
		t.Fatalf("stop queued run: %v", err)
	}
	waitStatus(t, e, canceled.ID, core.RunStopped)
	waitStatus(t, e, next.ID, core.RunQueued)

	fp.Release()
	waitStatus(t, e, first.ID, core.RunDone)
	waitStatus(t, e, next.ID, core.RunDone)
	prompts := fp.Prompts()
	if len(prompts) != 2 || !strings.Contains(prompts[1], "next request") {
		t.Fatalf("provider prompts after queued cancellation = %v, want first and next request", prompts)
	}
	if got := fp.MaxConcurrentCalls(); got != 1 {
		t.Fatalf("maximum concurrent provider calls = %d, want 1", got)
	}
}

func TestAgentQueuedRunsResumeInFIFOOrderAcrossDMs(t *testing.T) {
	fp := &fakeProvider{chunks: []string{"reply"}, release: make(chan struct{})}
	e := newEnv(t, fp, 1000000, 3)
	defer fp.Release()
	ctx := context.Background()
	secondConversation, err := e.svc.CreateDM(ctx, owner, e.agentID, "")
	if err != nil {
		t.Fatalf("create second dm: %v", err)
	}
	thirdConversation, err := e.svc.CreateDM(ctx, owner, e.agentID, "")
	if err != nil {
		t.Fatalf("create third dm: %v", err)
	}

	_, first, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "first request", "key-fifo-first")
	if err != nil {
		t.Fatalf("send first: %v", err)
	}
	waitPromptCount(t, fp, 1)
	_, second, err := e.svc.SendMessage(ctx, owner, secondConversation.ID, "second request", "key-fifo-second")
	if err != nil {
		t.Fatalf("send second: %v", err)
	}
	_, third, err := e.svc.SendMessage(ctx, owner, thirdConversation.ID, "third request", "key-fifo-third")
	if err != nil {
		t.Fatalf("send third: %v", err)
	}
	if second.Status != core.RunQueued || third.Status != core.RunQueued {
		t.Fatalf("queued runs = %s, %s; want both queued", second.Status, third.Status)
	}

	fp.Release()
	waitStatus(t, e, first.ID, core.RunDone)
	waitStatus(t, e, second.ID, core.RunDone)
	waitStatus(t, e, third.ID, core.RunDone)
	prompts := fp.Prompts()
	if len(prompts) != 3 ||
		!strings.Contains(prompts[0], "first request") ||
		!strings.Contains(prompts[1], "second request") ||
		!strings.Contains(prompts[2], "third request") {
		t.Fatalf("provider prompt order = %v, want first, second, third", prompts)
	}
	if got := fp.MaxConcurrentCalls(); got != 1 {
		t.Fatalf("maximum concurrent provider calls = %d, want 1", got)
	}
}

func TestFailedAgentRunReleasesSlotForNextDM(t *testing.T) {
	fp := &fakeProvider{failure: errors.New("provider failed"), release: make(chan struct{})}
	e := newEnv(t, fp, 1000000, 3)
	defer fp.Release()
	ctx := context.Background()
	secondConversation, err := e.svc.CreateDM(ctx, owner, e.agentID, "")
	if err != nil {
		t.Fatalf("create second dm: %v", err)
	}

	_, first, err := e.svc.SendMessage(ctx, owner, e.conv.ID, "first request", "key-failed-first")
	if err != nil {
		t.Fatalf("send first: %v", err)
	}
	waitPromptCount(t, fp, 1)
	_, queued, err := e.svc.SendMessage(ctx, owner, secondConversation.ID, "second request", "key-failed-second")
	if err != nil {
		t.Fatalf("send second: %v", err)
	}
	if queued.Status != core.RunQueued {
		t.Fatalf("second run = %s, want queued behind the active agent run", queued.Status)
	}

	fp.Release()
	waitStatus(t, e, first.ID, core.RunFailed)
	waitStatus(t, e, queued.ID, core.RunFailed)
	prompts := fp.Prompts()
	if len(prompts) != 2 || !strings.Contains(prompts[1], "second request") {
		t.Fatalf("provider prompts after failure = %v, want first and second request", prompts)
	}
	if got := fp.MaxConcurrentCalls(); got != 1 {
		t.Fatalf("maximum concurrent provider calls = %d, want 1", got)
	}
}
