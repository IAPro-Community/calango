-- Named secrets sealed by internal/vault (F1-T10, ADR 0012).
-- Values are envelopes only: no plain-text column exists. key_version names the master
-- key that wrapped the data key. Times are RFC 3339 UTC text, as in the other tables.

CREATE TABLE vault_items (
    owner_id    TEXT NOT NULL CHECK (length(owner_id) > 0),
    name        TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 63),
    ciphertext  TEXT NOT NULL CHECK (length(ciphertext) > 0),
    key_version INTEGER NOT NULL CHECK (key_version >= 1),
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    PRIMARY KEY (owner_id, name)
);
