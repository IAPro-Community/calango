-- Node-wide switches. One row; the CHECK keeps it a singleton.
CREATE TABLE node_state (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    global_paused INTEGER NOT NULL DEFAULT 0 CHECK (global_paused IN (0, 1)),
    updated_at TEXT NOT NULL
);

INSERT INTO node_state (id, global_paused, updated_at)
VALUES (1, 0, '1970-01-01T00:00:00.000Z');
