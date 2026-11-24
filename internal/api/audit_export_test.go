package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"calango/internal/core"
	"calango/internal/store"
)

func appendTwoAuditEvents(t *testing.T, ctx context.Context, database *store.Store) {
	t.Helper()
	at := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	for i, action := range []string{"vault.set", "vault.delete"} {
		ev := core.AuditEvent{
			ID: strings.Repeat(string(rune('1'+i)), 26), At: at.Add(time.Duration(i) * time.Second),
			ActorKind: "owner", ActorID: testOwner, Action: action,
			Subject: "vault:api-key", Facts: map[string]any{"name": "api-key"},
		}
		if err := database.AppendAudit(ctx, ev); err != nil {
			t.Fatalf("append %s: %v", action, err)
		}
	}
}

func TestAuditExportJSONLStartsWithCheckpointThenLinkedEntries(t *testing.T) {
	ctx := context.Background()
	h, database := newTestEnv(t)
	appendTwoAuditEvents(t, ctx, database)

	rec := call(t, h, http.MethodGet, "/v1/audit/export", testToken, "")
	expectStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Content-Type"); got != "application/x-ndjson" {
		t.Fatalf("content type = %q, want application/x-ndjson", got)
	}
	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want checkpoint plus two entries:\n%s", len(lines), rec.Body.String())
	}
	var cp auditExportCheckpoint
	if err := json.Unmarshal([]byte(lines[0]), &cp); err != nil || cp.Type != "checkpoint" || cp.EntryCount != 2 {
		t.Fatalf("first line = %s (err %v), want checkpoint with 2 entries", lines[0], err)
	}
	var first, second auditExportEntry
	if err := json.Unmarshal([]byte(lines[1]), &first); err != nil {
		t.Fatalf("decode first entry: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[2]), &second); err != nil {
		t.Fatalf("decode second entry: %v", err)
	}
	if first.PrevHash != "" || second.PrevHash != first.EntryHash || second.EntryHash != cp.HeadEntryHash {
		t.Fatalf("entries do not link to the checkpoint head: first %+v second %+v head %q", first, second, cp.HeadEntryHash)
	}
	if string(second.Facts) != `{"name":"api-key"}` {
		t.Fatalf("facts = %s, want the name only", second.Facts)
	}
}

func TestAuditExportCSVHasHeaderAndEntries(t *testing.T) {
	ctx := context.Background()
	h, database := newTestEnv(t)
	appendTwoAuditEvents(t, ctx, database)

	rec := call(t, h, http.MethodGet, "/v1/audit/export?format=csv", testToken, "")
	expectStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/csv") {
		t.Fatalf("content type = %q, want text/csv", got)
	}
	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "seq,id,at,actor_kind") {
		t.Fatalf("csv = %q, want header plus two entries", rec.Body.String())
	}
}

func TestAuditExportRejectsUnknownFormat(t *testing.T) {
	h := newTestHandler(t)
	expectStatus(t, call(t, h, http.MethodGet, "/v1/audit/export?format=xml", testToken, ""), http.StatusBadRequest)
}

func TestAuditExportIsReadScoped(t *testing.T) {
	h := newTestHandler(t)
	read := pairDevice(t, h, issuePairingCode(t, h, "read"))
	approveOnly := pairDevice(t, h, issuePairingCode(t, h, "approve_only"))

	expectStatus(t, call(t, h, http.MethodGet, "/v1/audit/export", read.Token, ""), http.StatusOK)
	expectStatus(t, call(t, h, http.MethodGet, "/v1/audit/export", approveOnly.Token, ""), http.StatusForbidden)
}
