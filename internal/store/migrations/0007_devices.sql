-- Paired devices and single-use pairing codes (F1-T15).
-- Only SHA-256 digests of tokens and codes are stored. Times are RFC 3339 UTC text,
-- the same layout as the other tables. Expired codes are purged when new codes are issued.

CREATE TABLE pairing_codes (
    digest      TEXT PRIMARY KEY CHECK (length(digest) = 64),
    owner_id    TEXT NOT NULL CHECK (length(owner_id) > 0),
    scope       TEXT NOT NULL CHECK (scope IN ('full', 'approve_only', 'read')),
    expires_at  TEXT NOT NULL,
    used_at     TEXT
);

CREATE TABLE devices (
    id            TEXT PRIMARY KEY CHECK (length(id) = 26),
    owner_id      TEXT NOT NULL CHECK (length(owner_id) > 0),
    name          TEXT NOT NULL CHECK (length(trim(name)) > 0),
    platform      TEXT NOT NULL CHECK (platform IN ('desktop', 'ios', 'android', 'web')),
    scope         TEXT NOT NULL CHECK (scope IN ('full', 'approve_only', 'read')),
    token_digest  TEXT NOT NULL UNIQUE CHECK (length(token_digest) = 64),
    created_at    TEXT NOT NULL,
    last_seen_at  TEXT,
    revoked_at    TEXT
);

CREATE INDEX devices_owner_id_idx ON devices (owner_id);
