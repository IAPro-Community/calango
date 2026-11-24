package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPreMigrationSnapshotRestoresAndAppliesPendingMigration(t *testing.T) {
	ctx := context.Background()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	if len(migrations) < 2 {
		t.Fatalf("got %d migrations; need a pending migration to test restore", len(migrations))
	}

	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "calango.db")
	db, err := sql.Open("sqlite", fileDSN(dbPath))
	if err != nil {
		t.Fatalf("open pre-migration database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := applyMigrations(ctx, db, migrations[:len(migrations)-1]); err != nil {
		t.Fatalf("apply pre-upgrade migrations: %v", err)
	}
	const agentID = "00000000000000000000000001"
	if _, err := db.ExecContext(ctx, `INSERT INTO agents
		(id, owner_id, name, token_limit_month, created_at, updated_at)
		VALUES (?, 'owner', 'Backup subject', 1000, '2026-11-23T00:00:00Z', '2026-11-23T00:00:00Z')`, agentID); err != nil {
		t.Fatalf("seed pre-upgrade agent: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close pre-upgrade database: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "calango.db")
	backup, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read offline snapshot: %v", err)
	}
	if err := os.WriteFile(backupPath, backup, 0o600); err != nil {
		t.Fatalf("write offline snapshot: %v", err)
	}

	upgraded, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("upgrade database after snapshot: %v", err)
	}
	if err := upgraded.Close(); err != nil {
		t.Fatalf("close upgraded database: %v", err)
	}

	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		if err := os.Remove(dbPath + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("remove upgraded database file %q: %v", suffix, err)
		}
	}
	if err := os.WriteFile(dbPath, backup, 0o600); err != nil {
		t.Fatalf("restore pre-migration snapshot: %v", err)
	}

	restored, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("reopen restored database and apply pending migration: %v", err)
	}
	defer func() { _ = restored.Close() }()
	var name string
	var maestroEnabled int
	if err := restored.db.QueryRowContext(ctx, `SELECT name, maestro_enabled FROM agents WHERE id = ?`, agentID).Scan(&name, &maestroEnabled); err != nil {
		t.Fatalf("read restored agent after migration: %v", err)
	}
	if name != "Backup subject" || maestroEnabled != 0 {
		t.Fatalf("restored agent = (%q, %d), want (%q, 0)", name, maestroEnabled, "Backup subject")
	}
}
