-- Provider chain per agent (ADR 0016). Accounts are named per owner; an agent
-- lists an ordered chain of (account, model) pairs. Secrets never live here.

CREATE TABLE provider_accounts (
    id TEXT PRIMARY KEY NOT NULL CHECK (length(id) = 26),
    owner_id TEXT NOT NULL CHECK (length(owner_id) > 0),
    kind TEXT NOT NULL CHECK (length(kind) > 0),
    label TEXT NOT NULL CHECK (length(label) BETWEEN 1 AND 32),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (owner_id, kind, label)
);

CREATE TABLE agent_providers (
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    account_id TEXT NOT NULL REFERENCES provider_accounts(id) ON DELETE RESTRICT,
    model TEXT NOT NULL CHECK (length(trim(model)) > 0),
    PRIMARY KEY (agent_id, position)
);

CREATE INDEX agent_providers_account_id_idx ON agent_providers(account_id);

-- Agents that used the single harness keep working: their harness becomes the
-- "default" account of the same kind (for codex: the shared Codex home).
INSERT INTO provider_accounts (id, owner_id, kind, label, created_at, updated_at)
SELECT lower(hex(randomblob(13))), owner_id, harness, 'default', created_at, created_at
FROM agents
WHERE harness IN ('codex', 'anthropic')
GROUP BY owner_id, harness;

INSERT INTO agent_providers (agent_id, position, account_id, model)
SELECT a.id, 0, acc.id, a.model
FROM agents a
JOIN provider_accounts acc ON acc.owner_id = a.owner_id AND acc.kind = a.harness AND acc.label = 'default'
WHERE a.model <> '';

ALTER TABLE agents DROP COLUMN harness;
ALTER TABLE agents DROP COLUMN model;

-- SQLite cannot alter a CHECK constraint, so run_steps is rebuilt to accept
-- provider_attempt steps.
CREATE TABLE run_steps_new (
    id TEXT PRIMARY KEY NOT NULL CHECK (length(id) = 26),
    run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    seq INTEGER NOT NULL CHECK (seq >= 0),
    kind TEXT NOT NULL CHECK (kind IN ('thought', 'tool_call', 'tool_result', 'text', 'provider_attempt')),
    payload_json TEXT NOT NULL CHECK (json_valid(payload_json)),
    created_at TEXT NOT NULL,
    UNIQUE (run_id, seq)
);

INSERT INTO run_steps_new (id, run_id, seq, kind, payload_json, created_at)
SELECT id, run_id, seq, kind, payload_json, created_at FROM run_steps;

DROP TABLE run_steps;
ALTER TABLE run_steps_new RENAME TO run_steps;
