-- Memories: facts, preferences and decisions the agent keeps across runs (F1-T11).
-- Deletion is logical (deleted_at) and takes effect on the next run; rows are removed
-- physically after the retention period. Deleting an agent deletes its agent-scoped memories.
-- Times are RFC 3339 UTC text, the same layout as the other tables.

CREATE TABLE memories (
    id          TEXT PRIMARY KEY CHECK (length(id) = 26),
    owner_id    TEXT NOT NULL CHECK (length(owner_id) > 0),
    scope       TEXT NOT NULL CHECK (scope IN ('account', 'agent')),
    agent_id    TEXT REFERENCES agents(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL CHECK (kind IN ('fact', 'preference', 'decision')),
    text        TEXT NOT NULL CHECK (length(trim(text)) > 0),
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    deleted_at  TEXT,
    CHECK ((scope = 'account') = (agent_id IS NULL))
);

CREATE INDEX memories_owner_deleted_idx ON memories (owner_id, deleted_at);
