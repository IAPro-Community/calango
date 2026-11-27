package store

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()

	database, err := OpenMemory(context.Background())
	if err != nil {
		t.Fatalf("OpenMemory() error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return database
}

func TestOpenMemoryAppliesSchemaAndEnforcesConstraints(t *testing.T) {
	database := openTestStore(t)
	rows, err := database.db.QueryContext(context.Background(), `
		SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatalf("query schema tables: %v", err)
	}
	gotTables := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan schema table: %v", err)
		}
		gotTables = append(gotTables, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate schema tables: %v", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close schema rows: %v", err)
	}
	wantTables := []string{"account_leases", "agent_providers", "agent_tools", "agents", "approval_rules", "approvals", "audit_checkpoints", "audit_log", "connectors", "conversation_members", "conversations", "devices", "memories", "messages", "node_state", "pairing_codes", "provider_accounts", "routine_runs", "routines", "run_steps", "runs", "schema_migrations", "usage", "vault_items"}
	if strings.Join(gotTables, ",") != strings.Join(wantTables, ",") {
		t.Fatalf("schema tables = %v, want %v", gotTables, wantTables)
	}

	ctx := context.Background()
	agentID := strings.Repeat("a", 26)
	if _, err := database.db.ExecContext(ctx, `
		INSERT INTO agents(id, owner_id, name, token_limit_month, created_at, updated_at)
		VALUES (?, 'owner', 'Assistant', 1000, '2026-11-23T00:00:00Z', '2026-11-23T00:00:00Z')`, agentID); err != nil {
		t.Fatalf("insert valid agent: %v", err)
	}
	if _, err := database.db.ExecContext(ctx, `
		INSERT INTO agents(id, owner_id, name, token_limit_month, created_at, updated_at)
		VALUES (?, 'owner', 'No budget', 0, '2026-11-23T00:00:00Z', '2026-11-23T00:00:00Z')`, strings.Repeat("b", 26)); err == nil {
		t.Fatal("agent with a zero monthly token limit was accepted")
	}

	conversationID := strings.Repeat("c", 26)
	messageID := strings.Repeat("d", 26)
	if _, err := database.db.ExecContext(ctx, `
		INSERT INTO conversations(id, kind, created_at) VALUES (?, 'dm', '2026-11-23T00:00:00Z')`, conversationID); err != nil {
		t.Fatalf("insert conversation: %v", err)
	}
	if _, err := database.db.ExecContext(ctx, `
		INSERT INTO messages(id, conversation_id, author_kind, author_id, body, created_at)
		VALUES (?, ?, 'agent', ?, 'Hello', '2026-11-23T00:00:00Z')`, messageID, conversationID, agentID); err != nil {
		t.Fatalf("insert message: %v", err)
	}
	if _, err := database.db.ExecContext(ctx, `
		INSERT INTO messages(id, conversation_id, author_kind, author_id, body, created_at)
		VALUES (?, ?, 'agent', ?, 'Orphan', '2026-11-23T00:00:00Z')`, strings.Repeat("e", 26), strings.Repeat("f", 26), agentID); err == nil {
		t.Fatal("message with a missing conversation was accepted")
	}

	if _, err := database.db.ExecContext(ctx, `
		INSERT INTO usage(id, at, agent_id, tokens_in, tokens_out, cost_micros, cost_status)
		VALUES (?, '2026-11-23T00:00:00Z', ?, 0, 0, 0, 'unknown')`, strings.Repeat("g", 26), agentID); err == nil {
		t.Fatal("unknown usage cost represented as zero was accepted")
	}
	if _, err := database.db.ExecContext(ctx, `
		INSERT INTO usage(id, at, agent_id, tokens_in, tokens_out, cost_micros, cost_status)
		VALUES (?, '2026-11-23T00:00:00Z', ?, 0, 0, NULL, 'unknown')`, strings.Repeat("h", 26), agentID); err != nil {
		t.Fatalf("insert usage with unknown cost: %v", err)
	}

	auditID := strings.Repeat("i", 26)
	if _, err := database.db.ExecContext(ctx, `
		INSERT INTO audit_log(id, at, actor_kind, action, subject, facts_json)
		VALUES (?, '2026-11-23T00:00:00Z', 'person', 'test', 'agent', '{}')`, auditID); err != nil {
		t.Fatalf("insert audit record: %v", err)
	}
	if _, err := database.db.ExecContext(ctx, `UPDATE audit_log SET action = 'changed' WHERE id = ?`, auditID); err == nil {
		t.Fatal("audit record update was accepted")
	}
	if _, err := database.db.ExecContext(ctx, `DELETE FROM audit_log WHERE id = ?`, auditID); err == nil {
		t.Fatal("audit record deletion was accepted")
	}
}

func TestOpenFileEnablesWALAndRestrictsPermissions(t *testing.T) {
	path := t.TempDir() + string(os.PathSeparator) + "calango.db"
	database, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	var journalMode string
	if err := database.db.QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("query journal mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal mode = %q, want wal", journalMode)
	}
	var foreignKeys int
	if err := database.db.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("query foreign keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat database file: %v", err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("database permissions = %04o, want 0600", got)
		}
	}
}
