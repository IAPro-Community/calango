-- Canonical identity keys and monotonic fencing survive daemon restarts.
-- Do not cascade-delete released records: their fence is still authoritative.
CREATE TABLE account_leases (
    owner_id TEXT NOT NULL CHECK (length(owner_id) > 0),
    provider TEXT NOT NULL CHECK (length(provider) > 0),
    account_id TEXT NOT NULL CHECK (length(account_id) > 0),
    holder TEXT NOT NULL,
    fence INTEGER NOT NULL CHECK (fence > 0),
    revision INTEGER NOT NULL CHECK (revision >= fence),
    expires_at TEXT NOT NULL,
    CHECK ((holder = '') = (expires_at = '')),
    PRIMARY KEY (owner_id, provider, account_id)
);
