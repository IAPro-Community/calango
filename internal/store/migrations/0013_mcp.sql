-- MCP connectors and per-agent tool grants (F1-T12).
-- A connector is one MCP server: stdio (command and args) or http (url). Connectors
-- hold no secrets. A grant is read or write; "none" is the absence of a row.

CREATE TABLE connectors (
    id         TEXT PRIMARY KEY NOT NULL CHECK (length(id) = 26),
    owner_id   TEXT NOT NULL CHECK (length(owner_id) > 0),
    name       TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 24),
    transport  TEXT NOT NULL CHECK (transport IN ('stdio', 'http')),
    command    TEXT NOT NULL DEFAULT '',
    args       TEXT NOT NULL DEFAULT '[]',
    url        TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK ((transport = 'stdio' AND length(command) > 0) OR (transport = 'http' AND length(url) > 0)),
    UNIQUE (owner_id, name)
);

CREATE TABLE agent_tools (
    agent_id     TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    connector_id TEXT NOT NULL REFERENCES connectors(id) ON DELETE CASCADE,
    access       TEXT NOT NULL CHECK (access IN ('read', 'write')),
    PRIMARY KEY (agent_id, connector_id)
);
