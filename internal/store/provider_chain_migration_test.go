package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
)

type chainRow struct {
	position int
	kind     string
	model    string
}

// chainOf returns the provider chain of agentID in position order.
func chainOf(t *testing.T, db *sql.DB, agentID string) []chainRow {
	t.Helper()
	rows, err := db.Query(`SELECT p.position, a.kind, p.model
		FROM agent_providers p JOIN provider_accounts a ON a.id = p.account_id
		WHERE p.agent_id = ? ORDER BY p.position`, agentID)
	if err != nil {
		t.Fatalf("query chain: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []chainRow
	for rows.Next() {
		var r chainRow
		if err := rows.Scan(&r.position, &r.kind, &r.model); err != nil {
			t.Fatalf("scan chain: %v", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate chain: %v", err)
	}
	return out
}

// TestProviderChainMigrationTurnsLegacyHarnessIntoChain upgrades a database that
// holds agents with the single harness and model columns. Migration 0010 must
// give each agent with a model a one-entry chain on its harness's default account.
func TestProviderChainMigrationTurnsLegacyHarnessIntoChain(t *testing.T) {
	ctx := context.Background()
	all, err := loadMigrations()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "legacy.db")+"?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := applyMigrations(ctx, db, all[:9]); err != nil {
		t.Fatalf("apply 0001-0009: %v", err)
	}
	const now = "2025-03-01T12:00:00Z"
	legacy := []struct{ id, owner, harness, model string }{
		{"00000000000000000000000001", "owner-a", "codex", "gpt-x"},
		{"00000000000000000000000002", "owner-a", "anthropic", "claude-x"},
		{"00000000000000000000000003", "owner-b", "codex", ""},
	}
	for _, a := range legacy {
		if _, err := db.ExecContext(ctx, `INSERT INTO agents
			(id, owner_id, name, harness, model, token_limit_month, created_at, updated_at)
			VALUES (?, ?, 'Calango', ?, ?, 1000, ?, ?)`, a.id, a.owner, a.harness, a.model, now, now); err != nil {
			t.Fatalf("insert legacy agent: %v", err)
		}
	}

	if err := applyMigrations(ctx, db, all); err != nil {
		t.Fatalf("apply 0010: %v", err)
	}

	var columns int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('agents') WHERE name IN ('harness', 'model')`).Scan(&columns); err != nil {
		t.Fatalf("read agents columns: %v", err)
	}
	if columns != 0 {
		t.Fatalf("agents still has %d harness/model columns", columns)
	}

	if got, want := chainOf(t, db, legacy[0].id), []chainRow{{0, "codex", "gpt-x"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("codex agent chain = %v, want %v", got, want)
	}
	if got, want := chainOf(t, db, legacy[1].id), []chainRow{{0, "anthropic", "claude-x"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("anthropic agent chain = %v, want %v", got, want)
	}
	// An agent that never had a model cannot name one, so it gets no chain and
	// must be configured again before it can run.
	if got := chainOf(t, db, legacy[2].id); len(got) != 0 {
		t.Fatalf("agent without model chain = %v, want none", got)
	}

	var accounts int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM provider_accounts WHERE label = 'default'`).Scan(&accounts); err != nil {
		t.Fatalf("count default accounts: %v", err)
	}
	if accounts != 3 {
		t.Fatalf("default accounts = %d, want one per owner and kind (3)", accounts)
	}
}
