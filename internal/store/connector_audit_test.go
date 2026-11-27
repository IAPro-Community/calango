package store

import (
	"context"
	"strings"
	"testing"

	"calango/internal/core"
)

func TestConnectorGrantsAndAuditAreAtomic(t *testing.T) {
	ctx := context.Background()
	st, err := OpenMemory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	agent := strings.Repeat("a", 26)
	insertTestAgent(t, st, agent, "owner")
	conn := testConnector(strings.Repeat("c", 26), "owner", "notes")
	if err := st.CreateConnector(ctx, conn); err != nil {
		t.Fatal(err)
	}
	grant := core.AgentTool{AgentID: agent, ConnectorID: conn.ID, Access: core.AccessRead}
	if err := st.SetAgentTool(ctx, grant); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(ctx, `CREATE TRIGGER test_reject_audit BEFORE INSERT ON audit_log
		BEGIN SELECT RAISE(ABORT, 'injected audit failure'); END`); err != nil {
		t.Fatal(err)
	}
	grant.Access = core.AccessWrite
	if err := st.SetAgentTool(ctx, grant); err == nil {
		t.Fatal("grant change without audit succeeded")
	}
	grants, err := st.ListAgentTools(ctx, agent)
	if err != nil || len(grants) != 1 || grants[0].Access != core.AccessRead {
		t.Fatalf("grant changed despite audit failure: %+v, %v", grants, err)
	}
	if _, err := st.db.ExecContext(ctx, "DROP TRIGGER test_reject_audit"); err != nil {
		t.Fatal(err)
	}
	if err := st.RemoveAgentTool(ctx, agent, conn.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteConnector(ctx, "owner", conn.ID); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"connector.created", "connector.grant_set", "connector.grant_revoked", "connector.deleted"} {
		events, err := st.ListAudit(ctx, action)
		if err != nil || len(events) != 1 {
			t.Fatalf("audit %s = %v, %v", action, events, err)
		}
		if events[0].ActorID != "owner" || events[0].Subject != conn.ID {
			t.Fatalf("unexpected audit authority for %s", action)
		}
		for _, key := range []string{"args", "url", "command", "secret", "output"} {
			if _, ok := events[0].Facts[key]; ok {
				t.Fatalf("audit contains forbidden field %s", key)
			}
		}
	}
	verified, err := st.VerifyAudit(ctx)
	if err != nil || verified.FirstBreak != nil || verified.EntriesChecked != 4 {
		t.Fatalf("audit verification = %+v, %v", verified, err)
	}
}
