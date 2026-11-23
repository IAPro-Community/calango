-- Tamper-evident chain for audit_log (F1-T18, ADR 0011: derived from vstorm-co/agenticos
-- backend/app/core/audit.py). Each entry stores the previous entry's hash and its own,
-- so editing, reordering or deleting an entry from the middle breaks the chain. The
-- checkpoint records the head so that dropping the newest entries is detected too.
-- Rows written before this migration have no entry_hash; verification reports them.
-- The checkpoint row can only move forward and cannot be deleted.

ALTER TABLE audit_log ADD COLUMN prev_hash TEXT;
ALTER TABLE audit_log ADD COLUMN entry_hash TEXT NOT NULL DEFAULT '';

CREATE TABLE audit_checkpoints (
    id              INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    max_seq         INTEGER NOT NULL CHECK (max_seq >= 1),
    entry_count     INTEGER NOT NULL CHECK (entry_count >= 1),
    head_entry_hash TEXT NOT NULL CHECK (length(head_entry_hash) = 64)
);

CREATE TRIGGER audit_checkpoints_reject_delete
BEFORE DELETE ON audit_checkpoints
BEGIN
    SELECT RAISE(ABORT, 'audit_checkpoints is append-only');
END;

CREATE TRIGGER audit_checkpoints_reject_rewind
BEFORE UPDATE ON audit_checkpoints
WHEN NEW.max_seq < OLD.max_seq OR NEW.entry_count < OLD.entry_count
BEGIN
    SELECT RAISE(ABORT, 'audit checkpoint cannot move backwards');
END;
