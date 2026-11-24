package approvals_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"calango/internal/core"
)

type ran struct {
	ownerID    string
	approvalID string
	action     core.Action
}

func connectorWrite(args string) core.Action {
	return core.Action{Kind: core.ActionConnectorWrite, Target: "notes/create", Args: json.RawMessage(args)}
}

// waitReceipt polls until the approval has an outcome, the way a caller waits for
// a background executor.
func waitReceipt(t *testing.T, e *env, id string) core.Approval {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		a, err := e.st.GetApproval(context.Background(), owner, id)
		if err != nil {
			t.Fatalf("get approval: %v", err)
		}
		if a.Outcome != "" {
			return a
		}
		if time.Now().After(deadline) {
			t.Fatalf("approval %s has no receipt; status %s", id, a.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestDecisionRunsExecutorWithTheApprovedSnapshot(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	got := make(chan ran, 1)
	e.svc.SetExecutor(core.ActionConnectorWrite, func(_ context.Context, ownerID, approvalID string, action core.Action) error {
		got <- ran{ownerID: ownerID, approvalID: approvalID, action: action}
		return nil
	})

	a := e.request(connectorWrite(`{"title":"groceries"}`))
	if _, err := e.svc.Decide(ctx, owner, a.ID, "dev-1", true, false); err != nil {
		t.Fatalf("decide: %v", err)
	}
	select {
	case r := <-got:
		if r.ownerID != owner || r.approvalID != a.ID {
			t.Fatalf("executor got owner %q approval %q", r.ownerID, r.approvalID)
		}
		if string(r.action.Args) != `{"title":"groceries"}` {
			t.Fatalf("executor ran args %s, not the approved snapshot", r.action.Args)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("executor did not run after approval")
	}
	if receipt := waitReceipt(t, e, a.ID); receipt.Outcome != core.OutcomeCompleted {
		t.Fatalf("receipt outcome = %s, want completed", receipt.Outcome)
	}
}

func TestRuleApprovalRunsExecutorWithoutACard(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	got := make(chan string, 2)
	e.svc.SetExecutor(core.ActionConnectorWrite, func(_ context.Context, _ string, approvalID string, _ core.Action) error {
		got <- approvalID
		return nil
	})

	first := e.request(connectorWrite(`{"title":"a"}`))
	if _, err := e.svc.Decide(ctx, owner, first.ID, "dev-1", true, true); err != nil {
		t.Fatalf("decide with rule: %v", err)
	}
	select {
	case <-got:
	case <-time.After(5 * time.Second):
		t.Fatal("first approval did not run")
	}
	waitReceipt(t, e, first.ID)

	second := e.request(connectorWrite(`{"title":"b"}`))
	if second.Status != core.ApprovalApproved {
		t.Fatalf("second status = %s, want auto-approved", second.Status)
	}
	select {
	case id := <-got:
		if id != second.ID {
			t.Fatalf("executor ran %s, want the auto-approved %s", id, second.ID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("auto-approved action did not run")
	}
	waitReceipt(t, e, second.ID)
}
