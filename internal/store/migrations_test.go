package store

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestMigrationsRejectChangesAfterApplication(t *testing.T) {
	database := openTestStore(t)
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations() error = %v", err)
	}
	version := len(migrations) + 1
	original := testMigration(version, fmt.Sprintf("%04d_probe.sql", version), "CREATE TABLE migration_probe (id INTEGER PRIMARY KEY);")
	migrations = append(migrations, original)
	if err := applyMigrations(context.Background(), database.db, migrations); err != nil {
		t.Fatalf("apply original migration: %v", err)
	}

	changed := testMigration(version, original.name, "CREATE TABLE migration_probe (id INTEGER PRIMARY KEY, value TEXT);")
	migrations[len(migrations)-1] = changed
	if err := applyMigrations(context.Background(), database.db, migrations); err == nil {
		t.Fatal("changed migration was accepted")
	}
}

func TestMigrationFailureRollsBackSchemaAndLedger(t *testing.T) {
	database := openTestStore(t)
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations() error = %v", err)
	}
	version := len(migrations) + 1
	broken := testMigration(version, fmt.Sprintf("%04d_broken.sql", version), strings.Join([]string{
		"CREATE TABLE migration_rollback_probe (id INTEGER PRIMARY KEY);",
		"CREATE TABLE migration_rollback_probe (id INTEGER PRIMARY KEY);",
	}, "\n"))
	migrations = append(migrations, broken)
	if err := applyMigrations(context.Background(), database.db, migrations); err == nil {
		t.Fatal("invalid migration succeeded")
	}

	var tableCount int
	if err := database.db.QueryRowContext(context.Background(), `
		SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'migration_rollback_probe'`).Scan(&tableCount); err != nil {
		t.Fatalf("query rollback table: %v", err)
	}
	if tableCount != 0 {
		t.Fatalf("rollback table count = %d, want 0", tableCount)
	}
	var migrationCount int
	if err := database.db.QueryRowContext(context.Background(), `
		SELECT count(*) FROM schema_migrations WHERE version = ?`, version).Scan(&migrationCount); err != nil {
		t.Fatalf("query migration ledger: %v", err)
	}
	if migrationCount != 0 {
		t.Fatalf("failed migration ledger count = %d, want 0", migrationCount)
	}
}

func testMigration(version int, name, sql string) migration {
	return migration{
		version:  version,
		name:     name,
		checksum: migrationChecksum([]byte(sql)),
		sql:      sql,
	}
}
