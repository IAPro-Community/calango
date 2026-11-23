package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"calango/internal/agents"
	"calango/internal/approvals"
	"calango/internal/core"
	"calango/internal/devices"
	"calango/internal/memory"
	"calango/internal/store"
	"calango/internal/usage"
)

const (
	testToken    = "test-token-0123456789"
	testOwner    = "owner-1"
	testDeviceID = "desk-1"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	database, err := store.OpenMemory(context.Background())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	now := func() time.Time { return time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC) }
	agentSvc, err := agents.NewService(database, now, core.NewID)
	if err != nil {
		t.Fatalf("agents service: %v", err)
	}
	memorySvc, err := memory.NewService(database, now, core.NewID)
	if err != nil {
		t.Fatalf("memory service: %v", err)
	}
	approvalSvc, err := approvals.NewService(database, time.Hour, now, core.NewID)
	if err != nil {
		t.Fatalf("approvals service: %v", err)
	}
	devicesSvc, err := devices.NewService(database, now, core.NewID)
	if err != nil {
		t.Fatalf("devices service: %v", err)
	}
	usageSvc, err := usage.NewService(database)
	if err != nil {
		t.Fatalf("usage service: %v", err)
	}
	h, err := NewHandler(Config{
		Agents:    agentSvc,
		Memories:  memorySvc,
		Approvals: approvalSvc,
		Devices:   devicesSvc,
		Usage:     usageSvc,
		Token:     testToken,
		OwnerID:   testOwner,
		DeviceID:  testDeviceID,
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	return h
}

// call sends one request. An empty token sends no Authorization header.
func call(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	return body.Error.Code
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, want, rec.Body.String())
	}
}

func TestRoutesRequireTheLocalBearerToken(t *testing.T) {
	h := newTestHandler(t)

	expectStatus(t, call(t, h, http.MethodGet, "/v1/agents", "", ""), http.StatusUnauthorized)
	rec := call(t, h, http.MethodGet, "/v1/agents", "wrong-token", "")
	expectStatus(t, rec, http.StatusUnauthorized)
	if got := errorCode(t, rec); got != "unauthorized" {
		t.Fatalf("code = %q, want unauthorized", got)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("401 must carry a WWW-Authenticate challenge")
	}
	expectStatus(t, call(t, h, http.MethodGet, "/v1/agents", testToken, ""), http.StatusOK)

	health := call(t, h, http.MethodGet, "/healthz", "", "")
	expectStatus(t, health, http.StatusOK)
	if !strings.Contains(health.Body.String(), `"ok"`) {
		t.Fatalf("healthz body = %s", health.Body.String())
	}
}

func TestAgentLifecycleThroughHTTP(t *testing.T) {
	h := newTestHandler(t)

	created := call(t, h, http.MethodPost, "/v1/agents", testToken,
		`{"name":"  Calango  ","token_limit_month":100000}`)
	expectStatus(t, created, http.StatusCreated)
	var agent struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		RunOn     string `json:"run_on"`
		Approval  string `json:"approval_level"`
		Paused    bool   `json:"paused"`
		TokensUse int64  `json:"tokens_used_month"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &agent); err != nil {
		t.Fatalf("decode agent: %v", err)
	}
	if agent.ID == "" || agent.Name != "Calango" {
		t.Fatalf("created agent = %+v; name must be trimmed and id set", agent)
	}
	if agent.RunOn != "pc" || agent.Approval != "sensitive" {
		t.Fatalf("defaults run_on=%q approval_level=%q, want pc and sensitive", agent.RunOn, agent.Approval)
	}

	path := "/v1/agents/" + agent.ID
	expectStatus(t, call(t, h, http.MethodGet, path, testToken, ""), http.StatusOK)

	replaced := call(t, h, http.MethodPut, path, testToken,
		`{"name":"Calango 2","token_limit_month":200000,"run_on":"vps"}`)
	expectStatus(t, replaced, http.StatusOK)
	if !strings.Contains(replaced.Body.String(), `"name":"Calango 2"`) {
		t.Fatalf("replace did not change name: %s", replaced.Body.String())
	}

	list := call(t, h, http.MethodGet, "/v1/agents", testToken, "")
	expectStatus(t, list, http.StatusOK)
	if strings.Count(list.Body.String(), `"id"`) != 1 {
		t.Fatalf("list = %s, want exactly one agent", list.Body.String())
	}

	expectStatus(t, call(t, h, http.MethodDelete, path, testToken, ""), http.StatusNoContent)
	gone := call(t, h, http.MethodGet, path, testToken, "")
	expectStatus(t, gone, http.StatusNotFound)
	if got := errorCode(t, gone); got != "not_found" {
		t.Fatalf("code = %q, want not_found", got)
	}
}

func TestAgentBodiesAreValidatedStrictly(t *testing.T) {
	h := newTestHandler(t)

	unknownField := call(t, h, http.MethodPost, "/v1/agents", testToken,
		`{"name":"A","token_limit_month":10,"bogus":true}`)
	expectStatus(t, unknownField, http.StatusBadRequest)
	if got := errorCode(t, unknownField); got != "invalid_input" {
		t.Fatalf("unknown field code = %q, want invalid_input", got)
	}

	noLimit := call(t, h, http.MethodPost, "/v1/agents", testToken, `{"name":"A"}`)
	expectStatus(t, noLimit, http.StatusBadRequest)

	trailing := call(t, h, http.MethodPost, "/v1/agents", testToken,
		`{"name":"A","token_limit_month":10} {"name":"B"}`)
	expectStatus(t, trailing, http.StatusBadRequest)
}

func TestMemoryLifecycleAndScopeOwnership(t *testing.T) {
	h := newTestHandler(t)

	created := call(t, h, http.MethodPost, "/v1/memories", testToken,
		`{"scope":"account","text":"Prefere respostas curtas"}`)
	expectStatus(t, created, http.StatusCreated)
	var m struct {
		ID      string `json:"id"`
		ScopeID string `json:"scope_id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode memory: %v", err)
	}
	if m.ScopeID != "" {
		t.Fatalf("account memory must not carry scope_id, got %q", m.ScopeID)
	}

	// An agent-scoped memory must name an agent this owner has.
	missingAgent := call(t, h, http.MethodPost, "/v1/memories", testToken,
		`{"scope":"agent","scope_id":"no-such-agent","text":"x"}`)
	expectStatus(t, missingAgent, http.StatusNotFound)

	badScope := call(t, h, http.MethodGet, "/v1/memories?scope=project", testToken, "")
	expectStatus(t, badScope, http.StatusBadRequest)

	expectStatus(t, call(t, h, http.MethodPatch, "/v1/memories/"+m.ID, testToken,
		`{"text":"Prefere respostas curtas e em português"}`), http.StatusOK)

	list := call(t, h, http.MethodGet, "/v1/memories?scope=account", testToken, "")
	expectStatus(t, list, http.StatusOK)
	if !strings.Contains(list.Body.String(), "em português") {
		t.Fatalf("list does not show the updated text: %s", list.Body.String())
	}

	expectStatus(t, call(t, h, http.MethodDelete, "/v1/memories/"+m.ID, testToken, ""), http.StatusNoContent)
	after := call(t, h, http.MethodGet, "/v1/memories", testToken, "")
	expectStatus(t, after, http.StatusOK)
	if strings.TrimSpace(after.Body.String()) != "[]" {
		t.Fatalf("deleted memory still listed: %s", after.Body.String())
	}
}

func TestApprovalsListAndDecisionErrors(t *testing.T) {
	h := newTestHandler(t)

	empty := call(t, h, http.MethodGet, "/v1/approvals", testToken, "")
	expectStatus(t, empty, http.StatusOK)
	if strings.TrimSpace(empty.Body.String()) != "[]" {
		t.Fatalf("empty list must be [], got %s", empty.Body.String())
	}

	expectStatus(t, call(t, h, http.MethodGet, "/v1/approvals?status=bogus", testToken, ""), http.StatusBadRequest)

	unknown := call(t, h, http.MethodPost, "/v1/approvals/none/decide", testToken, `{"decision":"approve"}`)
	expectStatus(t, unknown, http.StatusNotFound)

	badDecision := call(t, h, http.MethodPost, "/v1/approvals/none/decide", testToken, `{"decision":"maybe"}`)
	expectStatus(t, badDecision, http.StatusBadRequest)
}
