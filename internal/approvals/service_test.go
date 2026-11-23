package approvals_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"calango/internal/agents"
	"calango/internal/approvals"
	"calango/internal/core"
	"calango/internal/store"
)

const owner = "owner"

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

type env struct {
	t       *testing.T
	st      *store.Store
	clk     *clock
	svc     *approvals.Service
	agentID string
	runID   string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	st, err := store.OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	clk := &clock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	svc, err := approvals.NewService(st, approvals.DefaultTTL, clk.now, core.NewID)
	if err != nil {
		t.Fatalf("approvals service: %v", err)
	}
	ag, err := agents.NewService(st, clk.now, core.NewID)
	if err != nil {
		t.Fatalf("agents service: %v", err)
	}
	view, err := ag.Create(ctx, owner, core.Agent{Name: "Calango", Model: "m", Harness: "fake", TokenLimitMonth: 1000})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	conv, err := core.NewID(clk.t)
	if err != nil {
		t.Fatalf("id: %v", err)
	}
	if err := st.InsertConversation(ctx, core.Conversation{ID: conv, Kind: "dm", CreatedAt: clk.t}, view.ID); err != nil {
		t.Fatalf("insert conversation: %v", err)
	}
	runID, err := core.NewID(clk.t)
	if err != nil {
		t.Fatalf("id: %v", err)
	}
	started := clk.t
	if err := st.AdmitRun(ctx, nil, core.Run{
		ID: runID, AgentID: view.ID, ConversationID: conv, IdempotencyKey: runID,
		Trigger: "message", Status: core.RunStatus("running"), StartedAt: &started,
	}); err != nil {
		t.Fatalf("admit run: %v", err)
	}
	return &env{t: t, st: st, clk: clk, svc: svc, agentID: view.ID, runID: runID}
}

func (e *env) request(action core.Action) core.Approval {
	e.t.Helper()
	a, err := e.svc.Request(context.Background(), owner, e.agentID, e.runID, action)
	if err != nil {
		e.t.Fatalf("request: %v", err)
	}
	return a
}

func (e *env) count(action string) int {
	e.t.Helper()
	events, err := e.st.ListAudit(context.Background(), action)
	if err != nil {
		e.t.Fatalf("list audit: %v", err)
	}
	return len(events)
}

func email() core.Action {
	return core.Action{Kind: core.ActionSendEmail, Target: "ana@example.com", Args: json.RawMessage(`{"subject":"Oi","body":"texto"}`)}
}

func noRun(t *testing.T) func(context.Context, core.Action) error {
	return func(context.Context, core.Action) error {
		t.Fatal("action ran but must not")
		return nil
	}
}

func TestApprovedSnapshotRunsOnceAndRecordsReceipt(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	a := e.request(email())
	if a.Status != core.ApprovalPending || a.Seq != 1 {
		t.Fatalf("new request = status %s seq %d, want pending seq 1", a.Status, a.Seq)
	}

	if _, err := e.svc.Execute(ctx, owner, a.ID, email(), noRun(t)); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("execute while pending: err = %v, want ErrConflict", err)
	}
	if _, err := e.svc.Decide(ctx, owner, a.ID, "", true, false); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("decide without device: err = %v, want ErrInvalidInput", err)
	}
	if _, err := e.svc.Decide(ctx, owner, a.ID, "dev-1", true, false); err != nil {
		t.Fatalf("approve: %v", err)
	}

	var ran core.Action
	calls := 0
	got, err := e.svc.Execute(ctx, owner, a.ID, email(), func(_ context.Context, act core.Action) error {
		calls++
		ran = act
		return nil
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if calls != 1 || ran.Target != "ana@example.com" {
		t.Fatalf("run calls = %d target = %q, want 1 and ana@example.com", calls, ran.Target)
	}
	if got.Outcome != core.OutcomeCompleted || got.OutcomeReason != "" {
		t.Fatalf("receipt = %s (%q), want completed", got.Outcome, got.OutcomeReason)
	}
	if got.DecidedByDeviceID != "dev-1" {
		t.Fatalf("decided by = %q, want dev-1", got.DecidedByDeviceID)
	}
	if _, err := e.svc.Execute(ctx, owner, a.ID, email(), noRun(t)); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("second execute: err = %v, want ErrConflict", err)
	}
	for action, want := range map[string]int{"approval.requested": 1, "approval.decided": 1, "approval.executed": 1} {
		if n := e.count(action); n != want {
			t.Fatalf("audit %s = %d, want %d", action, n, want)
		}
	}
}

func TestFailedExecutionShowsReason(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	a := e.request(email())
	if _, err := e.svc.Decide(ctx, owner, a.ID, "dev-1", true, false); err != nil {
		t.Fatalf("approve: %v", err)
	}
	got, err := e.svc.Execute(ctx, owner, a.ID, email(), func(context.Context, core.Action) error {
		return errors.New("smtp refused the message")
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got.Status != core.ApprovalApproved || got.Outcome != core.OutcomeFailed || got.OutcomeReason != "smtp refused the message" {
		t.Fatalf("receipt = status %s outcome %s reason %q", got.Status, got.Outcome, got.OutcomeReason)
	}
}

func TestChangedActionIsRefusedAndRequestedAgain(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	a := e.request(email())
	if _, err := e.svc.Decide(ctx, owner, a.ID, "dev-1", true, false); err != nil {
		t.Fatalf("approve: %v", err)
	}

	changed := email()
	changed.Target = "outra@example.com"
	next, err := e.svc.Execute(ctx, owner, a.ID, changed, noRun(t))
	if !errors.Is(err, core.ErrDigestMismatch) {
		t.Fatalf("execute changed action: err = %v, want ErrDigestMismatch", err)
	}
	if next.Status != core.ApprovalPending || next.Seq != 2 || next.Action.Target != "outra@example.com" {
		t.Fatalf("new request = status %s seq %d target %q", next.Status, next.Seq, next.Action.Target)
	}
	original, err := e.svc.Get(ctx, owner, a.ID)
	if err != nil {
		t.Fatalf("get original: %v", err)
	}
	if original.Outcome != "" || original.Action.Target != "ana@example.com" {
		t.Fatalf("original approval changed: outcome %q target %q", original.Outcome, original.Action.Target)
	}
	if n := e.count("approval.blocked"); n != 1 {
		t.Fatalf("audit approval.blocked = %d, want 1", n)
	}
}

func TestAlwaysAllowSkipsCardButIsAudited(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	first := e.request(email())
	if _, err := e.svc.Decide(ctx, owner, first.ID, "dev-1", true, true); err != nil {
		t.Fatalf("approve and remember: %v", err)
	}

	second := e.request(email())
	if second.Status != core.ApprovalApproved {
		t.Fatalf("second request status = %s, want approved by rule", second.Status)
	}
	pending, err := e.svc.List(ctx, owner, core.ApprovalPending)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending cards = %d, want 0", len(pending))
	}
	if n := e.count("approval.auto_approved"); n != 1 {
		t.Fatalf("audit approval.auto_approved = %d, want 1", n)
	}
	if _, err := e.svc.Execute(ctx, owner, second.ID, email(), func(context.Context, core.Action) error { return nil }); err != nil {
		t.Fatalf("execute auto-approved: %v", err)
	}
}

func TestDenyIsFinal(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	a := e.request(email())
	if _, err := e.svc.Decide(ctx, owner, a.ID, "dev-1", false, true); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("remember a denial: err = %v, want ErrInvalidInput", err)
	}
	got, err := e.svc.Decide(ctx, owner, a.ID, "dev-1", false, false)
	if err != nil {
		t.Fatalf("deny: %v", err)
	}
	if got.Status != core.ApprovalDenied || got.Outcome != core.OutcomeNotRun {
		t.Fatalf("denied = status %s outcome %s", got.Status, got.Outcome)
	}
	if _, err := e.svc.Decide(ctx, owner, a.ID, "dev-1", true, false); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("approve after deny: err = %v, want ErrConflict", err)
	}
	if _, err := e.svc.Execute(ctx, owner, a.ID, email(), noRun(t)); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("execute denied: err = %v, want ErrConflict", err)
	}
}

func TestExpiredApprovalCannotBeDecided(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	a := e.request(email())
	e.clk.t = e.clk.t.Add(approvals.DefaultTTL)

	if _, err := e.svc.Decide(ctx, owner, a.ID, "dev-1", true, false); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("decide after expiry: err = %v, want ErrConflict", err)
	}
	got, err := e.svc.Get(ctx, owner, a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != core.ApprovalExpired || got.Outcome != core.OutcomeNotRun {
		t.Fatalf("expired = status %s outcome %s", got.Status, got.Outcome)
	}

	b := e.request(email())
	e.clk.t = e.clk.t.Add(approvals.DefaultTTL)
	n, err := e.svc.Sweep(ctx)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if n != 1 {
		t.Fatalf("swept = %d, want 1", n)
	}
	swept, err := e.svc.Get(ctx, owner, b.ID)
	if err != nil {
		t.Fatalf("get swept: %v", err)
	}
	if swept.Status != core.ApprovalExpired {
		t.Fatalf("swept status = %s, want expired", swept.Status)
	}
}

func TestOtherOwnersCannotSeeOrDecide(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	a := e.request(email())
	if _, err := e.svc.Get(ctx, "intruder", a.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("get by other owner: err = %v, want ErrNotFound", err)
	}
	if _, err := e.svc.Decide(ctx, "intruder", a.ID, "dev-x", true, false); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("decide by other owner: err = %v, want ErrNotFound", err)
	}
	if _, err := e.svc.Request(ctx, "intruder", e.agentID, e.runID, email()); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("request for other owner's agent: err = %v, want ErrNotFound", err)
	}
}

func TestNonSensitiveActionIsRejected(t *testing.T) {
	e := newEnv(t)
	_, err := e.svc.Request(context.Background(), owner, e.agentID, e.runID, core.Action{
		Kind: core.ActionKind("read_file"), Target: "notes.md",
	})
	if !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("request read_file: err = %v, want ErrInvalidInput", err)
	}
}
