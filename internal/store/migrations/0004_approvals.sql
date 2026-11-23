CREATE TABLE approvals (
    id TEXT PRIMARY KEY NOT NULL CHECK (length(id) = 26),
    run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE RESTRICT,
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE RESTRICT,
    seq INTEGER NOT NULL CHECK (seq > 0),
    action TEXT NOT NULL CHECK (length(action) > 0),
    target TEXT NOT NULL CHECK (length(target) > 0),
    args_json TEXT NOT NULL CHECK (json_valid(args_json)),
    action_digest TEXT NOT NULL CHECK (length(action_digest) = 64),
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'denied', 'expired')),
    outcome TEXT CHECK (outcome IN ('completed', 'failed', 'not_run')),
    outcome_reason TEXT,
    decided_by_device_id TEXT,
    decided_at TEXT,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (run_id, seq),
    CHECK (status <> 'pending' OR (decided_at IS NULL AND outcome IS NULL)),
    CHECK (outcome IS NULL OR status <> 'pending'),
    CHECK (outcome <> 'failed' OR length(outcome_reason) > 0)
);

CREATE INDEX approvals_status_idx ON approvals(status, expires_at);

CREATE TRIGGER approvals_snapshot_immutable
BEFORE UPDATE OF run_id, agent_id, seq, action, target, args_json, action_digest, expires_at, created_at ON approvals
BEGIN
    SELECT RAISE(ABORT, 'approval snapshot is immutable');
END;

CREATE TRIGGER approvals_status_is_final
BEFORE UPDATE OF status ON approvals
WHEN OLD.status <> NEW.status AND OLD.status <> 'pending'
BEGIN
    SELECT RAISE(ABORT, 'approval status is final');
END;

CREATE TRIGGER approvals_outcome_is_final
BEFORE UPDATE OF outcome ON approvals
WHEN OLD.outcome IS NOT NULL
BEGIN
    SELECT RAISE(ABORT, 'approval outcome is final');
END;

CREATE TABLE approval_rules (
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    action TEXT NOT NULL CHECK (length(action) > 0),
    decision TEXT NOT NULL CHECK (decision = 'always_allow'),
    created_at TEXT NOT NULL,
    PRIMARY KEY (agent_id, action)
);
