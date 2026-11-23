// Package store owns the node's SQLite database and applies its immutable migrations.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	maxOpenConnections = 4
	sqliteBusyTimeout  = "5000"
)

// Store is the persistent database for one Calango node.
type Store struct {
	db *sql.DB
}

// Open opens or creates a private SQLite database at path, enables WAL, and
// applies all pending migrations before returning.
func Open(ctx context.Context, path string) (*Store, error) {
	if path == ":memory:" {
		return OpenMemory(ctx)
	}
	if path == "" || strings.TrimSpace(path) != path {
		return nil, errors.New("database path must be non-empty and have no surrounding whitespace")
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o700); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	file, err := os.OpenFile(absolutePath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create database file: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close database file: %w", closeErr))
		}
		return nil, fmt.Errorf("restrict database file permissions: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close database file: %w", err)
	}

	return open(ctx, fileDSN(absolutePath), false)
}

// OpenMemory opens an isolated in-memory SQLite database for tests.
func OpenMemory(ctx context.Context) (*Store, error) {
	return open(ctx, "file::memory:?cache=shared", true)
}

func open(ctx context.Context, dsn string, memory bool) (*Store, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database DSN: %w", err)
	}
	query := parsed.Query()
	query.Add("_pragma", "foreign_keys(ON)")
	query.Add("_pragma", "busy_timeout("+sqliteBusyTimeout+")")
	query.Add("_pragma", "synchronous(NORMAL)")
	parsed.RawQuery = query.Encode()

	db, err := sql.Open("sqlite", parsed.String())
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConnections)
	db.SetMaxIdleConns(maxOpenConnections)

	closeWithError := func(cause error) (*Store, error) {
		if closeErr := db.Close(); closeErr != nil {
			cause = errors.Join(cause, fmt.Errorf("close SQLite database: %w", closeErr))
		}
		return nil, cause
	}

	if err := db.PingContext(ctx); err != nil {
		return closeWithError(fmt.Errorf("connect to SQLite database: %w", err))
	}
	if !memory {
		var journalMode string
		if err := db.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&journalMode); err != nil {
			return closeWithError(fmt.Errorf("enable SQLite WAL: %w", err))
		}
		if !strings.EqualFold(journalMode, "wal") {
			return closeWithError(fmt.Errorf("enable SQLite WAL: journal mode is %q", journalMode))
		}
	}

	if err := applyEmbeddedMigrations(ctx, db); err != nil {
		return closeWithError(fmt.Errorf("apply SQLite migrations: %w", err))
	}
	return &Store{db: db}, nil
}

func fileDSN(path string) string {
	path = filepath.ToSlash(path)
	if runtime.GOOS == "windows" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}

// Close releases database connections and SQLite file handles.
func (s *Store) Close() error {
	return s.db.Close()
}
