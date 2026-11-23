package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"calango/internal/core"
)

// approvalFixture inserts an agent, a conversation and a running run, and returns
// the run ID. Approvals need all three because of their foreign keys.
func approvalFixture(t *testing.T, st *Store) string {
	t.Helper()
	ctx := context.Background()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	agentID := strings.Repeat("a", 26)
	if _, err := st.db.ExecContext(ctx, `
		INSERT INTO agents(id, owner_id, name, token_limit_month, created_at, updated_at)
		VALUES (?, 'owner', 'Assistant', 1000, ?, ?)`, agentID, at.Format(usageTimeLayout), at.Format(usageTimeLayout)); err != nil {
		t.Fatalf("insert agent: %v", err)
	}
	convID := strings.Repeat("c", 26)
	if err := st.InsertConversation(ctx, core.Conversation{ID: convID, Kind: "dm", CreatedAt: at}, agentID); err != nil {
		t.Fatalf("insert conversation: %v", err)
	}
	runID := strings.Repeat("r", 26)
	if err := st.AdmitRun(ctx, nil, core.Run{
		ID: runID, AgentID: agentID, ConversationID: convID, IdempotencyKey: "key-1",
		Trigger: "message", Status: core.RunStatus("running"), StartedAt: &at,
	}); err != nil {
		t.Fatalf("admit run: %v", err)
	}
	return runID
}

func pendingApproval(runID string, at time.Time) core.Approval {
	return core.Approval{
		ID:        strings.Repeat("p", 26),
		RunID:     runID,
		AgentID:   strings.Repeat("a", 26),
		Seq:       1,
		Action:    core.Action{Kind: core.ActionDelete, Target: "file.txt", Args: []byte(`{"force":false}`)},
		Digest:    strings.Repeat("d", 64),
		Status:    core.ApprovalPending,
		ExpiresAt: at.Add(time.Hour),
		CreatedAt: at,
	}
}

func auditEvent(id string, at time.Time, action string) core.AuditEvent {
	return core.AuditEvent{ID: id, At: at, ActorKind: "system", Action: action, Subject: "s", Facts: map[string]any{"k": "v"}}
}

func TestApprovalSnapshotAndStatusAreEnforcedByTheDatabase(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	a := pendingApproval(approvalFixture(t, st), at)
	if err := st.InsertApproval(ctx, a, auditEvent(strings.Repeat("1", 26), at, "approval.requested")); err != nil {
		t.Fatalf("insert approval: %v", err)
	}

	if _, err := st.db.ExecContext(ctx, `UPDATE approvals SET args_json = '{"force":true}' WHERE id = ?`, a.ID); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("editing the snapshot: err = %v, want immutable", err)
	}

	if err := st.DecideApproval(ctx, ApprovalDecision{
		ID: a.ID, Status: core.ApprovalApproved, DeviceID: "dev-1", At: at,
		Audit: auditEvent(strings.Repeat("2", 26), at, "approval.decided"),
	}); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := st.db.ExecContext(ctx, `UPDATE approvals SET status = 'denied' WHERE id = ?`, a.ID); err == nil || !strings.Contains(err.Error(), "status is final") {
		t.Fatalf("changing a decided status: err = %v, want status is final", err)
	}

	if err := st.CompleteApproval(ctx, a.ID, core.OutcomeCompleted, "", auditEvent(strings.Repeat("3", 26), at, "approval.executed")); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if _, err := st.db.ExecContext(ctx, `UPDATE approvals SET outcome = 'failed', outcome_reason = 'late' WHERE id = ?`, a.ID); err == nil || !strings.Contains(err.Error(), "outcome is final") {
		t.Fatalf("changing a receipt: err = %v, want outcome is final", err)
	}
}

func TestDecideApprovalOnlyOnce(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	a := pendingApproval(approvalFixture(t, st), at)
	if err := st.InsertApproval(ctx, a, auditEvent(strings.Repeat("1", 26), at, "approval.requested")); err != nil {
		t.Fatalf("insert approval: %v", err)
	}
	deny := ApprovalDecision{
		ID: a.ID, Status: core.ApprovalDenied, Outcome: core.OutcomeNotRun, DeviceID: "dev-1", At: at,
		Audit: auditEvent(strings.Repeat("2", 26), at, "approval.decided"),
	}
	if err := st.DecideApproval(ctx, deny); err != nil {
		t.Fatalf("deny: %v", err)
	}
	deny.Audit.ID = strings.Repeat("4", 26)
	if err := st.DecideApproval(ctx, deny); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("second decision: err = %v, want ErrConflict", err)
	}

	events, err := st.ListAudit(ctx, "approval.decided")
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("decided audit entries = %d, want 1 (failed decision must not be audited)", len(events))
	}
}
