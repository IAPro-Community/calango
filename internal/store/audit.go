// Derived from vstorm-co/agenticos (Apache-2.0) at a4b0b3c: backend/app/core/audit.py
// (record_audit) and backend/app/services/audit.py (verify_chain). Modified.

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"calango/internal/audit"
	"calango/internal/core"
)

// insertAudit appends one audit event inside the caller's transaction and links it to the
// chain head, then advances the checkpoint. The head is read in the same transaction: under
// WAL, a writer whose snapshot is stale fails with SQLITE_BUSY instead of linking to the
// same predecessor twice, so the chain cannot fork.
func insertAudit(ctx context.Context, tx *sql.Tx, ev core.AuditEvent) error {
	facts, err := json.Marshal(ev.Facts)
	if err != nil {
		return fmt.Errorf("encode audit facts: %w", err)
	}
	fields := audit.Fields{
		At:        ev.At.UTC().Format(usageTimeLayout),
		ActorKind: ev.ActorKind,
		ActorID:   ev.ActorID,
		Action:    ev.Action,
		Subject:   ev.Subject,
		FactsJSON: string(facts),
	}
	prev, err := chainHead(ctx, tx)
	if err != nil {
		return err
	}
	hash, err := audit.Hash(prev, fields)
	if err != nil {
		return fmt.Errorf("hash audit %s: %w", ev.Action, err)
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO audit_log
		(id, at, actor_kind, actor_id, action, subject, facts_json, prev_hash, entry_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.ID, fields.At, ev.ActorKind, nullIfEmpty(ev.ActorID), ev.Action, ev.Subject,
		fields.FactsJSON, nullIfEmpty(prev), hash)
	if err != nil {
		return fmt.Errorf("append audit %s: %w", ev.Action, err)
	}
	seq, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("read audit sequence: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_checkpoints
		(id, max_seq, entry_count, head_entry_hash) VALUES (1, ?, 1, ?)
		ON CONFLICT(id) DO UPDATE SET
			max_seq = excluded.max_seq,
			entry_count = audit_checkpoints.entry_count + 1,
			head_entry_hash = excluded.head_entry_hash`, seq, hash); err != nil {
		return fmt.Errorf("advance audit checkpoint: %w", err)
	}
	return nil
}

// chainHead returns the hash of the newest entry, or "" when the chain is empty.
func chainHead(ctx context.Context, tx *sql.Tx) (string, error) {
	var head string
	err := tx.QueryRowContext(ctx,
		`SELECT entry_hash FROM audit_log ORDER BY rowid DESC LIMIT 1`).Scan(&head)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read audit head: %w", err)
	}
	return head, nil
}

// VerifyAudit recomputes the audit chain and reports the first break. The checkpoint and
// the entries are read in one read transaction, so an append committing in between cannot
// make an intact chain look truncated.
func (s *Store) VerifyAudit(ctx context.Context) (audit.Verification, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return audit.Verification{}, fmt.Errorf("begin audit verify: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	cp, err := loadCheckpoint(ctx, tx)
	if err != nil {
		return audit.Verification{}, err
	}
	records, err := loadChain(ctx, tx)
	if err != nil {
		return audit.Verification{}, err
	}
	return audit.Verify(records, cp), nil
}

func loadCheckpoint(ctx context.Context, tx *sql.Tx) (*audit.Checkpoint, error) {
	var cp audit.Checkpoint
	err := tx.QueryRowContext(ctx, `SELECT max_seq, entry_count, head_entry_hash
		FROM audit_checkpoints WHERE id = 1`).Scan(&cp.MaxSeq, &cp.EntryCount, &cp.HeadHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read audit checkpoint: %w", err)
	}
	return &cp, nil
}

func loadChain(ctx context.Context, tx *sql.Tx) ([]audit.Record, error) {
	rows, err := tx.QueryContext(ctx, `SELECT rowid, id, at, actor_kind, COALESCE(actor_id, ''),
		action, subject, facts_json, COALESCE(prev_hash, ''), entry_hash
		FROM audit_log ORDER BY rowid`)
	if err != nil {
		return nil, fmt.Errorf("read audit chain: %w", err)
	}
	defer rows.Close()
	var out []audit.Record
	for rows.Next() {
		var r audit.Record
		if err := rows.Scan(&r.Seq, &r.ID, &r.Fields.At, &r.Fields.ActorKind, &r.Fields.ActorID,
			&r.Fields.Action, &r.Fields.Subject, &r.Fields.FactsJSON, &r.PrevHash, &r.EntryHash); err != nil {
			return nil, fmt.Errorf("scan audit chain: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit chain: %w", err)
	}
	return out, nil
}
