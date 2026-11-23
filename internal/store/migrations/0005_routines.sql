-- Routines: scheduled work an agent performs on its own (F1-T09).
-- Times are RFC 3339 UTC text, the same layout as the other tables.

CREATE TABLE routines (
    id            TEXT PRIMARY KEY CHECK (length(id) = 26),
    owner_id      TEXT NOT NULL CHECK (length(owner_id) > 0),
    agent_id      TEXT NOT NULL REFERENCES agents(id),
    name          TEXT NOT NULL CHECK (length(trim(name)) > 0),
    prompt        TEXT NOT NULL CHECK (length(trim(prompt)) > 0),
    trigger_kind  TEXT NOT NULL CHECK (trigger_kind IN ('cron', 'interval')),
    schedule      TEXT NOT NULL CHECK (length(trim(schedule)) > 0),
    run_on        TEXT NOT NULL CHECK (run_on IN ('pc', 'vps', 'auto')),
    quiet_start   INTEGER CHECK (quiet_start IS NULL OR quiet_start BETWEEN 0 AND 1439),
    quiet_end     INTEGER CHECK (quiet_end IS NULL OR quiet_end BETWEEN 0 AND 1439),
    max_retries   INTEGER NOT NULL DEFAULT 0 CHECK (max_retries BETWEEN 0 AND 5),
    notify_on     TEXT NOT NULL DEFAULT 'failure' CHECK (notify_on IN ('always', 'news', 'failure')),
    paused        INTEGER NOT NULL DEFAULT 0 CHECK (paused IN (0, 1)),
    next_run_at   TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    CHECK ((quiet_start IS NULL) = (quiet_end IS NULL)),
    CHECK (quiet_start IS NULL OR quiet_start <> quiet_end)
);

CREATE INDEX routines_due_idx ON routines (paused, next_run_at);
CREATE INDEX routines_agent_idx ON routines (agent_id);

-- One row per scheduled time. UNIQUE(routine_id, occurrence_id) makes a redelivered
-- occurrence (after a restart or a retry) reuse the same row.
CREATE TABLE routine_runs (
    id              TEXT PRIMARY KEY CHECK (length(id) = 26),
    routine_id      TEXT NOT NULL REFERENCES routines(id) ON DELETE CASCADE,
    occurrence_id   TEXT NOT NULL CHECK (length(occurrence_id) > 0),
    scheduled_for   TEXT NOT NULL,
    run_id          TEXT REFERENCES runs(id),
    attempt         INTEGER NOT NULL DEFAULT 0 CHECK (attempt >= 0),
    status          TEXT NOT NULL CHECK (status IN ('pending', 'running', 'done', 'failed')),
    had_news        INTEGER CHECK (had_news IS NULL OR had_news IN (0, 1)),
    last_error      TEXT NOT NULL DEFAULT '',
    next_attempt_at TEXT NOT NULL,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    UNIQUE (routine_id, occurrence_id)
);

CREATE INDEX routine_runs_due_idx ON routine_runs (status, next_attempt_at);
