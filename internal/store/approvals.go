package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"calango/internal/core"
)

const approvalColumns = `id, run_id, agent_id, seq, action, target, args_json, action_digest,
	status, outcome, outcome_reason, decided_by_device_id, decided_at, expires_at, created_at`

// ApprovalDecision is one state transition with its audit entry, written in one
// transaction. Status must move away from pending; the trigger enforces it.
type ApprovalDecision struct {
	ID       string
	Status   core.ApprovalStatus
	Outcome  core.ApprovalOutcome // empty for approved: set later by CompleteApproval
	DeviceID string               // empty when the system decides (expiry)
	At       time.Time
	// RuleAgentID and RuleAction, when both set, store an "always allow" rule
	// in the same transaction.
	RuleAgentID string
	RuleAction  core.ActionKind
	Audit       core.AuditEvent
}

// InsertApproval stores a new approval and its audit entry in one transaction.
func (s *Store) InsertApproval(ctx context.Context, a core.Approval, ev core.AuditEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin approval insert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `INSERT INTO approvals (`+approvalColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.RunID, a.AgentID, a.Seq, string(a.Action.Kind), a.Action.Target, string(a.Action.Args),
		a.Digest, string(a.Status), nullIfEmpty(string(a.Outcome)), nullIfEmpty(a.OutcomeReason),
		nullIfEmpty(a.DecidedByDeviceID), timeValue(a.DecidedAt),
		a.ExpiresAt.UTC().Format(usageTimeLayout), a.CreatedAt.UTC().Format(usageTimeLayout)); err != nil {
		return fmt.Errorf("insert approval %s: %w", a.ID, err)
	}
	if err := insertAudit(ctx, tx, ev); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit approval insert: %w", err)
	}
	return nil
}

// GetApproval returns the approval only when its agent belongs to ownerID.
func (s *Store) GetApproval(ctx context.Context, ownerID, id string) (core.Approval, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+approvalColumns+` FROM approvals
		WHERE id = ? AND agent_id IN (SELECT id FROM agents WHERE owner_id = ?)`, id, ownerID)
	a, err := scanApproval(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Approval{}, fmt.Errorf("%w: approval %s", core.ErrNotFound, id)
	}
	if err != nil {
		return core.Approval{}, fmt.Errorf("read approval %s: %w", id, err)
	}
	return a, nil
}

// ListApprovals returns the owner's approvals, oldest first. An empty status
// returns every status.
func (s *Store) ListApprovals(ctx context.Context, ownerID string, status core.ApprovalStatus) ([]core.Approval, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+approvalColumns+` FROM approvals
		WHERE agent_id IN (SELECT id FROM agents WHERE owner_id = ?)
		AND (? = '' OR status = ?)
		ORDER BY created_at, id`, ownerID, string(status), string(status))
	if err != nil {
		return nil, fmt.Errorf("list approvals: %w", err)
	}
	return collectApprovals(rows)
}

// ListDueApprovals returns pending approvals whose expiry time has passed.
func (s *Store) ListDueApprovals(ctx context.Context, now time.Time) ([]core.Approval, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+approvalColumns+` FROM approvals
		WHERE status = 'pending' AND expires_at <= ? ORDER BY expires_at, id`,
		now.UTC().Format(usageTimeLayout))
	if err != nil {
		return nil, fmt.Errorf("list due approvals: %w", err)
	}
	return collectApprovals(rows)
}

// ListUnreceiptedApprovals returns approved approvals with no receipt, oldest
// first. Call it only at startup, while no executor is running.
func (s *Store) ListUnreceiptedApprovals(ctx context.Context) ([]core.Approval, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+approvalColumns+` FROM approvals
		WHERE status = 'approved' AND outcome IS NULL ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list approvals without receipt: %w", err)
	}
	return collectApprovals(rows)
}

// NextApprovalSeq returns the sequence number for the next approval of a run.
func (s *Store) NextApprovalSeq(ctx context.Context, runID string) (int64, error) {
	var next int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(seq), 0) + 1 FROM approvals WHERE run_id = ?`, runID).Scan(&next); err != nil {
		return 0, fmt.Errorf("next approval seq for run %s: %w", runID, err)
	}
	return next, nil
}

// AppendAudit writes one audit entry on its own. Use it only for events that do not
// change approval state; state changes write their audit entry in the same transaction.
func (s *Store) AppendAudit(ctx context.Context, ev core.AuditEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin audit append: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertAudit(ctx, tx, ev); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit audit append: %w", err)
	}
	return nil
}

// DecideApproval moves a pending approval to its decision, optionally stores an
// always-allow rule, and appends the audit entry, all in one transaction. It
// returns core.ErrConflict when the approval is no longer pending.
func (s *Store) DecideApproval(ctx context.Context, d ApprovalDecision) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin approval decision: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `UPDATE approvals
		SET status = ?, outcome = ?, decided_by_device_id = ?, decided_at = ?
		WHERE id = ? AND status = 'pending'`,
		string(d.Status), nullIfEmpty(string(d.Outcome)), nullIfEmpty(d.DeviceID),
		d.At.UTC().Format(usageTimeLayout), d.ID)
	if err != nil {
		return fmt.Errorf("decide approval %s: %w", d.ID, err)
	}
	if err := requireTransition(res, "approval "+d.ID+" is no longer pending"); err != nil {
		return err
	}
	if d.RuleAgentID != "" && d.RuleAction != "" {
		if _, err := tx.ExecContext(ctx, `INSERT INTO approval_rules (agent_id, action, decision, created_at)
			VALUES (?, ?, 'always_allow', ?)
			ON CONFLICT (agent_id, action) DO NOTHING`,
			d.RuleAgentID, string(d.RuleAction), d.At.UTC().Format(usageTimeLayout)); err != nil {
			return fmt.Errorf("insert approval rule: %w", err)
		}
	}
	if err := insertAudit(ctx, tx, d.Audit); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit approval decision: %w", err)
	}
	return nil
}

// CompleteApproval records the receipt of an approved action that has no
// outcome yet, with its audit entry. It returns core.ErrConflict otherwise.
func (s *Store) CompleteApproval(ctx context.Context, id string, outcome core.ApprovalOutcome, reason string, ev core.AuditEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin approval receipt: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `UPDATE approvals SET outcome = ?, outcome_reason = ?
		WHERE id = ? AND status = 'approved' AND outcome IS NULL`,
		string(outcome), nullIfEmpty(reason), id)
	if err != nil {
		return fmt.Errorf("record approval %s outcome: %w", id, err)
	}
	if err := requireTransition(res, "approved approval "+id+" without outcome"); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, ev); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit approval receipt: %w", err)
	}
	return nil
}

// HasRule reports whether the agent has an always-allow rule for the kind.
func (s *Store) HasRule(ctx context.Context, agentID string, kind core.ActionKind) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM approval_rules WHERE agent_id = ? AND action = ?`,
		agentID, string(kind)).Scan(&n); err != nil {
		return false, fmt.Errorf("read approval rule: %w", err)
	}
	return n > 0, nil
}

// ListAudit returns audit events for one action name, oldest first.
func (s *Store) ListAudit(ctx context.Context, action string) ([]core.AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, at, actor_kind, COALESCE(actor_id, ''),
		action, subject, facts_json FROM audit_log WHERE action = ? ORDER BY at, id`, action)
	if err != nil {
		return nil, fmt.Errorf("list audit %s: %w", action, err)
	}
	defer rows.Close()

	var out []core.AuditEvent
	for rows.Next() {
		var (
			ev    core.AuditEvent
			at    string
			facts string
		)
		if err := rows.Scan(&ev.ID, &at, &ev.ActorKind, &ev.ActorID, &ev.Action, &ev.Subject, &facts); err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}
		if ev.At, err = time.Parse(usageTimeLayout, at); err != nil {
			return nil, fmt.Errorf("parse audit time: %w", err)
		}
		if err := json.Unmarshal([]byte(facts), &ev.Facts); err != nil {
			return nil, fmt.Errorf("decode audit facts %s: %w", ev.ID, err)
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit: %w", err)
	}
	return out, nil
}

func collectApprovals(rows *sql.Rows) ([]core.Approval, error) {
	defer rows.Close()
	var out []core.Approval
	for rows.Next() {
		a, err := scanApproval(rows)
		if err != nil {
			return nil, fmt.Errorf("scan approval: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate approvals: %w", err)
	}
	return out, nil
}

func scanApproval(row rowScanner) (core.Approval, error) {
	var (
		a                                core.Approval
		kind, status, args               string
		outcome, reason, device, decided sql.NullString
		expires, created                 string
	)
	err := row.Scan(&a.ID, &a.RunID, &a.AgentID, &a.Seq, &kind, &a.Action.Target, &args, &a.Digest,
		&status, &outcome, &reason, &device, &decided, &expires, &created)
	if err != nil {
		return core.Approval{}, err
	}
	a.Action.Kind = core.ActionKind(kind)
	a.Action.Args = json.RawMessage(args)
	a.Status = core.ApprovalStatus(status)
	a.Outcome = core.ApprovalOutcome(outcome.String)
	a.OutcomeReason = reason.String
	a.DecidedByDeviceID = device.String
	if a.DecidedAt, err = optionalTime(decided); err != nil {
		return core.Approval{}, err
	}
	if a.ExpiresAt, err = time.Parse(usageTimeLayout, expires); err != nil {
		return core.Approval{}, fmt.Errorf("parse approval expiry: %w", err)
	}
	if a.CreatedAt, err = time.Parse(usageTimeLayout, created); err != nil {
		return core.Approval{}, fmt.Errorf("parse approval creation: %w", err)
	}
	return a, nil
}

// requireTransition reports a compare-and-set that matched no row. The approval
// exists (the caller checked its owner) but another decision got there first.
func requireTransition(res sql.Result, what string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("count affected rows for %s: %w", what, err)
	}
	if n == 0 {
		return fmt.Errorf("%w: %s", core.ErrConflict, what)
	}
	return nil
}
