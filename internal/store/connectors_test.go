package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"calango/internal/core"
)

func insertTestAgent(t *testing.T, st *Store, id, owner string) {
	t.Helper()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Format(usageTimeLayout)
	if _, err := st.db.ExecContext(context.Background(), `
		INSERT INTO agents(id, owner_id, name, token_limit_month, created_at, updated_at)
		VALUES (?, ?, 'Assistant', 1000, ?, ?)`, id, owner, at, at); err != nil {
		t.Fatalf("insert agent: %v", err)
	}
}

func testConnector(id, owner, name string) core.Connector {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return core.Connector{
		ID: id, OwnerID: owner, Name: name, Transport: core.TransportStdio,
		Command: "mcp-notes", Args: []string{"--root", "/notes"}, CreatedAt: at, UpdatedAt: at,
	}
}

func TestConnectorsAreOwnerScopedAndNamesAreUnique(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	c := testConnector(strings.Repeat("x", 26), "owner", "notes")
	if err := st.CreateConnector(ctx, c); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := st.GetConnector(ctx, "owner", c.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Command != "mcp-notes" || len(got.Args) != 2 || got.Args[1] != "/notes" {
		t.Fatalf("round trip lost fields: %+v", got)
	}
	if _, err := st.GetConnector(ctx, "someone-else", c.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("other owner read connector: %v", err)
	}

	dup := testConnector(strings.Repeat("y", 26), "owner", "notes")
	if err := st.CreateConnector(ctx, dup); !errors.Is(err, core.ErrConflict) {
		t.Fatalf("duplicate name err = %v, want conflict", err)
	}
	list, err := st.ListConnectors(ctx, "owner")
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %d, %v; want 1", len(list), err)
	}
}

func TestGrantsAreReplacedRemovedAndFollowTheirParents(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	agentID := strings.Repeat("a", 26)
	insertTestAgent(t, st, agentID, "owner")
	c := testConnector(strings.Repeat("x", 26), "owner", "notes")
	if err := st.CreateConnector(ctx, c); err != nil {
		t.Fatalf("create: %v", err)
	}

	grant := core.AgentTool{AgentID: agentID, ConnectorID: c.ID, Access: core.AccessRead}
	if err := st.SetAgentTool(ctx, grant); err != nil {
		t.Fatalf("set read: %v", err)
	}
	grant.Access = core.AccessWrite
	if err := st.SetAgentTool(ctx, grant); err != nil {
		t.Fatalf("replace with write: %v", err)
	}
	grants, err := st.ListAgentTools(ctx, agentID)
	if err != nil || len(grants) != 1 || grants[0].Access != core.AccessWrite {
		t.Fatalf("grants = %+v, %v; want one write grant", grants, err)
	}

	if err := st.DeleteConnector(ctx, "owner", c.ID); err != nil {
		t.Fatalf("delete connector: %v", err)
	}
	if grants, _ := st.ListAgentTools(ctx, agentID); len(grants) != 0 {
		t.Fatalf("deleting the connector kept grants: %+v", grants)
	}

	if err := st.CreateConnector(ctx, c); err != nil {
		t.Fatalf("recreate: %v", err)
	}
	if err := st.SetAgentTool(ctx, core.AgentTool{AgentID: agentID, ConnectorID: c.ID, Access: core.AccessRead}); err != nil {
		t.Fatalf("set again: %v", err)
	}
	if _, err := st.db.ExecContext(ctx, `DELETE FROM agents WHERE id = ?`, agentID); err != nil {
		t.Fatalf("delete agent: %v", err)
	}
	var n int
	if err := st.db.QueryRowContext(ctx, `SELECT count(*) FROM agent_tools`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("deleting the agent kept %d grants (%v)", n, err)
	}
}

func TestConnectorCheckRejectsAGrantWithoutTransportTarget(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	bad := testConnector(strings.Repeat("z", 26), "owner", "remote")
	bad.Transport, bad.Command = core.TransportHTTP, ""
	if err := st.CreateConnector(ctx, bad); err == nil {
		t.Fatal("http connector without url was stored")
	}
}
