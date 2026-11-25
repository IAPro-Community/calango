package approvals_test

import (
	"context"
	"errors"
	"testing"

	"calango/internal/approvals"
	"calango/internal/core"
	"calango/internal/store"
)

type receiptFailureStore struct {
	*store.Store
	fail bool
}

func (s *receiptFailureStore) CompleteApproval(ctx context.Context, id string, outcome core.ApprovalOutcome, reason string, ev core.AuditEvent) error {
	if s.fail {
		return errors.New("injected receipt write failure")
	}
	return s.Store.CompleteApproval(ctx, id, outcome, reason, ev)
}

func TestReceiptWriteFailureFencesExecutionUntilRestart(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	repo := &receiptFailureStore{Store: e.st, fail: true}
	svc, err := approvals.NewService(repo, approvals.DefaultTTL, e.clk.now, core.NewID)
	if err != nil {
		t.Fatal(err)
	}
	a := e.request(email())
	if _, err := svc.Decide(ctx, owner, a.ID, "dev-1", true, false); err != nil {
		t.Fatal(err)
	}
	calls := 0
	run := func(context.Context, core.Action) error {
		calls++
		return nil
	}
	if _, err := svc.Execute(ctx, owner, a.ID, email(), run); err == nil {
		t.Fatal("expected receipt failure")
	}
	repo.fail = false
	if _, err := svc.Execute(ctx, owner, a.ID, email(), run); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("retry error = %v, want conflict", err)
	}
	if calls != 1 {
		t.Fatalf("external executions = %d, want 1", calls)
	}

	restarted, err := approvals.NewService(repo, approvals.DefaultTTL, e.clk.now, core.NewID)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := restarted.Reconcile(ctx); err != nil || n != 1 {
		t.Fatalf("reconcile = (%d, %v), want (1, nil)", n, err)
	}
	got, err := restarted.Get(ctx, owner, a.ID)
	if err != nil || got.Outcome != core.OutcomeFailed {
		t.Fatalf("receipt = (%+v, %v), want failed/unknown", got, err)
	}
	if _, err := restarted.Execute(ctx, owner, a.ID, email(), run); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("retry after restart = %v, want conflict", err)
	}
	if calls != 1 {
		t.Fatalf("external executions after restart = %d, want 1", calls)
	}
}
