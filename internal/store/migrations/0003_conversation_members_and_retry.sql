-- Conversation members: the agent partner of a DM, and later the people and
-- agents of a channel. member_id is not a foreign key because it points at
-- different tables depending on member_kind.
CREATE TABLE conversation_members (
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE RESTRICT,
    member_kind TEXT NOT NULL CHECK (member_kind IN ('person', 'agent')),
    member_id TEXT NOT NULL CHECK (length(member_id) > 0),
    joined_at TEXT NOT NULL,
    PRIMARY KEY (conversation_id, member_kind, member_id)
);

-- Retry with instructions: a new run that points at the stopped or failed run
-- it redoes, and carries the extra direction.
ALTER TABLE runs ADD COLUMN instructions TEXT NOT NULL DEFAULT '';
ALTER TABLE runs ADD COLUMN retry_of TEXT REFERENCES runs(id) ON DELETE SET NULL;

CREATE INDEX runs_status_idx ON runs(status);
