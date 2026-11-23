// Derived from vstorm-co/agenticos (Apache-2.0) at a4b0b3c: backend/app/core/audit.py and
// backend/app/services/audit.py (verify_chain). Scenarios rewritten for SQLite. Modified.

package store

import (
	"context"
	"strings"
	"testing"
	"time"
)

func appendThree(t *testing.T, st *Store) {
	t.Helper()
	ctx := context.Background()
	at := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	for i, action := range []string{"approval.requested", "approval.decided", "device.revoked"} {
		if err := st.AppendAudit(ctx, auditEvent(strings.Repeat(string(rune('1'+i)), 26), at.Add(time.Duration(i)*time.Second), action)); err != nil {
			t.Fatalf("append %s: %v", action, err)
		}
	}
}

func TestAuditChainVerifiesAfterAppends(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	appendThree(t, st)
	got, err := st.VerifyAudit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.FirstBreak != nil || got.EntriesChecked != 3 {
		t.Fatalf("verify after appends: got %+v, want 3 entries and no break", got)
	}
}

func TestAuditEditDetectedWhenGuardTriggerIsDropped(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	appendThree(t, st)
	// The append-only triggers stop an application edit; an operator with direct database
	// access can drop them. The chain has to catch that.
	if _, err := st.db.ExecContext(ctx, `DROP TRIGGER audit_log_reject_update`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(ctx,
		`UPDATE audit_log SET facts_json = '{"decision":"denied"}' WHERE rowid = 2`); err != nil {
		t.Fatal(err)
	}
	got, err := st.VerifyAudit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.FirstBreak == nil || got.FirstBreak.Seq != 2 {
		t.Fatalf("edited row: got %+v, want break at seq 2", got.FirstBreak)
	}
}

func TestAuditTruncationDetectedByCheckpoint(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	appendThree(t, st)
	if _, err := st.db.ExecContext(ctx, `DROP TRIGGER audit_log_reject_delete`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(ctx, `DELETE FROM audit_log WHERE rowid = 3`); err != nil {
		t.Fatal(err)
	}
	got, err := st.VerifyAudit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.FirstBreak == nil || !strings.Contains(got.FirstBreak.Reason, "tail truncated") {
		t.Fatalf("truncated tail: got %+v, want checkpoint break", got.FirstBreak)
	}
}

func TestAuditCheckpointCannotMoveBackwards(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	appendThree(t, st)
	_, err = st.db.ExecContext(ctx, `UPDATE audit_checkpoints SET max_seq = 1, entry_count = 1`)
	if err == nil || !strings.Contains(err.Error(), "cannot move backwards") {
		t.Fatalf("rewind checkpoint: got %v, want trigger abort", err)
	}
}
