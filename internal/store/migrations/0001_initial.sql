CREATE TABLE agents (
    id TEXT PRIMARY KEY NOT NULL CHECK (length(id) = 26),
    owner_id TEXT NOT NULL CHECK (length(owner_id) > 0),
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    role TEXT NOT NULL DEFAULT '',
    instructions TEXT NOT NULL DEFAULT '',
    harness TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    reasoning_effort TEXT CHECK (reasoning_effort IN ('low', 'medium', 'high', 'max')),
    run_on TEXT NOT NULL DEFAULT 'pc' CHECK (run_on IN ('pc', 'vps', 'auto')),
    token_limit_month INTEGER NOT NULL CHECK (token_limit_month > 0),
    approval_level TEXT NOT NULL DEFAULT 'sensitive' CHECK (approval_level IN ('always', 'sensitive', 'trusted')),
    paused INTEGER NOT NULL DEFAULT 0 CHECK (paused IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX agents_owner_id_idx ON agents(owner_id);

CREATE TABLE conversations (
    id TEXT PRIMARY KEY NOT NULL CHECK (length(id) = 26),
    kind TEXT NOT NULL CHECK (kind IN ('dm', 'channel')),
    title TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE messages (
    id TEXT PRIMARY KEY NOT NULL CHECK (length(id) = 26),
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE RESTRICT,
    thread_id TEXT,
    author_kind TEXT NOT NULL CHECK (author_kind IN ('person', 'agent', 'system')),
    author_id TEXT NOT NULL CHECK (length(author_id) > 0),
    body TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX messages_conversation_created_idx ON messages(conversation_id, created_at, id);

CREATE TABLE runs (
    id TEXT PRIMARY KEY NOT NULL CHECK (length(id) = 26),
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE RESTRICT,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE RESTRICT,
    source_message_id TEXT REFERENCES messages(id) ON DELETE SET NULL,
    idempotency_key TEXT NOT NULL UNIQUE CHECK (length(idempotency_key) > 0),
    trigger TEXT NOT NULL CHECK (trigger IN ('message', 'routine', 'task')),
    node_id TEXT,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'done', 'failed', 'stopped', 'waiting_approval')),
    tokens_in INTEGER NOT NULL DEFAULT 0 CHECK (tokens_in >= 0),
    tokens_out INTEGER NOT NULL DEFAULT 0 CHECK (tokens_out >= 0),
    cost_micros INTEGER CHECK (cost_micros IS NULL OR cost_micros >= 0),
    cost_status TEXT NOT NULL DEFAULT 'unknown' CHECK (cost_status IN ('known', 'estimated', 'unknown')),
    started_at TEXT,
    ended_at TEXT,
    error_code TEXT,
    CHECK (
        (cost_status = 'unknown' AND cost_micros IS NULL)
        OR (cost_status IN ('known', 'estimated') AND cost_micros IS NOT NULL)
    )
);

CREATE INDEX runs_agent_started_idx ON runs(agent_id, started_at, id);
CREATE INDEX runs_conversation_status_idx ON runs(conversation_id, status);

CREATE TABLE run_steps (
    id TEXT PRIMARY KEY NOT NULL CHECK (length(id) = 26),
    run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    seq INTEGER NOT NULL CHECK (seq >= 0),
    kind TEXT NOT NULL CHECK (kind IN ('thought', 'tool_call', 'tool_result', 'text')),
    payload_json TEXT NOT NULL CHECK (json_valid(payload_json)),
    created_at TEXT NOT NULL,
    UNIQUE (run_id, seq)
);

CREATE TABLE usage (
    id TEXT PRIMARY KEY NOT NULL CHECK (length(id) = 26),
    at TEXT NOT NULL,
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE RESTRICT,
    routine_id TEXT,
    project_id TEXT,
    node_id TEXT,
    provider TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    tokens_in INTEGER NOT NULL DEFAULT 0 CHECK (tokens_in >= 0),
    tokens_out INTEGER NOT NULL DEFAULT 0 CHECK (tokens_out >= 0),
    cost_micros INTEGER CHECK (cost_micros IS NULL OR cost_micros >= 0),
    cost_status TEXT NOT NULL CHECK (cost_status IN ('known', 'estimated', 'unknown')),
    pricing_version TEXT,
    CHECK (
        (cost_status = 'unknown' AND cost_micros IS NULL)
        OR (cost_status IN ('known', 'estimated') AND cost_micros IS NOT NULL)
    )
);

CREATE INDEX usage_at_agent_idx ON usage(at, agent_id);
CREATE INDEX usage_routine_at_idx ON usage(routine_id, at);

CREATE TABLE audit_log (
    id TEXT PRIMARY KEY NOT NULL CHECK (length(id) = 26),
    at TEXT NOT NULL,
    actor_kind TEXT NOT NULL CHECK (length(actor_kind) > 0),
    actor_id TEXT,
    action TEXT NOT NULL CHECK (length(action) > 0),
    subject TEXT NOT NULL CHECK (length(subject) > 0),
    facts_json TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(facts_json))
);

CREATE INDEX audit_log_at_idx ON audit_log(at, id);

CREATE TRIGGER audit_log_reject_update
BEFORE UPDATE ON audit_log
BEGIN
    SELECT RAISE(ABORT, 'audit_log is append-only');
END;

CREATE TRIGGER audit_log_reject_delete
BEFORE DELETE ON audit_log
BEGIN
    SELECT RAISE(ABORT, 'audit_log is append-only');
END;
