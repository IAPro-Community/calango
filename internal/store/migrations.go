package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

var migrationNamePattern = regexp.MustCompile(`^([0-9]{4})_([a-z0-9]+(?:_[a-z0-9]+)*)\.sql$`)

type migration struct {
	version  int
	name     string
	checksum string
	sql      string
}

type appliedMigration struct {
	name     string
	checksum string
}

func loadMigrations() ([]migration, error) {
	files, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return nil, fmt.Errorf("list migration files: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, errors.New("no migration files found")
	}

	migrations := make([]migration, 0, len(files))
	for index, file := range files {
		name := path.Base(file)
		matches := migrationNamePattern.FindStringSubmatch(name)
		if matches == nil {
			return nil, fmt.Errorf("invalid migration filename %q", name)
		}
		version, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("parse migration version in %q: %w", name, err)
		}
		if version != index+1 {
			return nil, fmt.Errorf("migration %q has version %d; expected %04d", name, version, index+1)
		}

		contents, err := migrationFiles.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", name, err)
		}
		if strings.TrimSpace(string(contents)) == "" {
			return nil, fmt.Errorf("migration %q is empty", name)
		}
		migrations = append(migrations, migration{
			version:  version,
			name:     name,
			checksum: migrationChecksum(contents),
			sql:      string(contents),
		})
	}
	return migrations, nil
}

func migrationChecksum(contents []byte) string {
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:])
}

func applyEmbeddedMigrations(ctx context.Context, db *sql.DB) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	return applyMigrations(ctx, db, migrations)
}

func applyMigrations(ctx context.Context, db *sql.DB, migrations []migration) error {
	if len(migrations) == 0 {
		return errors.New("no migrations to apply")
	}
	for index, current := range migrations {
		if current.version != index+1 {
			return fmt.Errorf("migration %q has version %d; expected %04d", current.name, current.version, index+1)
		}
		if current.name == "" || strings.TrimSpace(current.sql) == "" {
			return fmt.Errorf("migration %04d must have a name and SQL", current.version)
		}
		if current.checksum != migrationChecksum([]byte(current.sql)) {
			return fmt.Errorf("migration %q has an invalid checksum", current.name)
		}
	}

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY CHECK (version > 0),
			name TEXT NOT NULL UNIQUE,
			checksum TEXT NOT NULL CHECK (length(checksum) = 64),
			applied_at TEXT NOT NULL
		)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	applied, err := readAppliedMigrations(ctx, db)
	if err != nil {
		return err
	}
	for version, record := range applied {
		if version > len(migrations) {
			return fmt.Errorf("database has unknown migration version %04d", version)
		}
		current := migrations[version-1]
		if record.name != current.name || record.checksum != current.checksum {
			return fmt.Errorf("applied migration %q differs from its immutable source", current.name)
		}
	}

	for _, current := range migrations {
		if _, ok := applied[current.version]; ok {
			continue
		}
		if err := applyMigration(ctx, db, current); err != nil {
			return err
		}
		applied[current.version] = appliedMigration{name: current.name, checksum: current.checksum}
	}
	return nil
}

func readAppliedMigrations(ctx context.Context, db *sql.DB) (map[int]appliedMigration, error) {
	rows, err := db.QueryContext(ctx, `SELECT version, name, checksum FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("read migration ledger: %w", err)
	}
	applied := make(map[int]appliedMigration)
	for rows.Next() {
		var version int
		var record appliedMigration
		if err := rows.Scan(&version, &record.name, &record.checksum); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan migration ledger: %w", err)
		}
		if version != len(applied)+1 {
			_ = rows.Close()
			return nil, fmt.Errorf("migration ledger has a gap before version %04d", version)
		}
		applied[version] = record
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate migration ledger: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close migration ledger rows: %w", err)
	}
	return applied, nil
}

func applyMigration(ctx context.Context, db *sql.DB, current migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %q: %w", current.name, err)
	}

	result, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO schema_migrations(version, name, checksum, applied_at)
		VALUES (?, ?, ?, ?)`,
		current.version,
		current.name,
		current.checksum,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return rollbackMigration(tx, current.name, fmt.Errorf("reserve migration %q: %w", current.name, err))
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return rollbackMigration(tx, current.name, fmt.Errorf("check migration reservation %q: %w", current.name, err))
	}
	if rowsAffected == 0 {
		var name, checksum string
		if err := tx.QueryRowContext(ctx, `
			SELECT name, checksum FROM schema_migrations WHERE version = ?`, current.version,
		).Scan(&name, &checksum); err != nil {
			return rollbackMigration(tx, current.name, fmt.Errorf("read concurrent migration %q: %w", current.name, err))
		}
		if name != current.name || checksum != current.checksum {
			return rollbackMigration(tx, current.name, fmt.Errorf("applied migration %q differs from its immutable source", current.name))
		}
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			return fmt.Errorf("finish concurrent migration %q: %w", current.name, err)
		}
		return nil
	}
	if rowsAffected != 1 {
		return rollbackMigration(tx, current.name, fmt.Errorf("migration %q reserved %d ledger rows", current.name, rowsAffected))
	}

	if _, err := tx.ExecContext(ctx, current.sql); err != nil {
		return rollbackMigration(tx, current.name, fmt.Errorf("execute migration %q: %w", current.name, err))
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %q: %w", current.name, err)
	}
	return nil
}

func rollbackMigration(tx *sql.Tx, name string, cause error) error {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return errors.Join(cause, fmt.Errorf("roll back migration %q: %w", name, err))
	}
	return cause
}
