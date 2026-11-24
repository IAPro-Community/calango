package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"calango/internal/agents"
	"calango/internal/approvals"
	"calango/internal/auth"
	"calango/internal/conversations"
	"calango/internal/core"
	"calango/internal/devices"
	"calango/internal/events"
	"calango/internal/gateway"
	"calango/internal/memory"
	"calango/internal/providers"
	"calango/internal/routines"
	"calango/internal/store"
	"calango/internal/tools"
	"calango/internal/usage"
	"calango/internal/vault"
)

const (
	testToken    = "test-token-0123456789"
	testOwner    = "owner-1"
	testDeviceID = "desk-1"
)

// testAccountID is the default Anthropic account seeded for testOwner.
var testAccountID = "acct" + strings.Repeat("0", 22)

// testChain is the agent JSON fragment that chains the seeded account.
var testChain = fmt.Sprintf(`"providers":[{"account_id":%q,"model":"claude-test"}]`, testAccountID)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	h, _ := newTestEnv(t)
	return h
}

// newTestEnv returns the handler and the store behind it, so a test can read the audit trail.
func newTestEnv(t *testing.T) (http.Handler, *store.Store) {
	t.Helper()
	return newTestEnvWith(t, nil)
}

// newTestEnvWith is newTestEnv with trusted proxy ranges for the pairing limit.
func newTestEnvWith(t *testing.T, trusted []netip.Prefix) (http.Handler, *store.Store) {
	h, database, _ := newTestEnvWithApproval(t, trusted)
	return h, database
}

// newTestEnvWithApproval returns the handler, store and approvals service.
func newTestEnvWithApproval(t *testing.T, trusted []netip.Prefix) (http.Handler, *store.Store, *approvals.Service) {
	t.Helper()
	gw := &blockingGateway{done: make(chan struct{})}
	return newTestEnvWithGateway(t, trusted, gw, func() { close(gw.done) })
}

func newTestEnvWithGateway(t *testing.T, trusted []netip.Prefix, gw conversations.Gateway, releaseGateway func()) (http.Handler, *store.Store, *approvals.Service) {
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
	keys, err := vault.NewMasterKeys(map[int][]byte{1: bytes.Repeat([]byte{7}, 32)})
	if err != nil {
		t.Fatalf("master keys: %v", err)
	}
	vaultSvc, err := vault.NewService(database, keys, testOwner, database, now, core.NewID)
	if err != nil {
		t.Fatalf("vault service: %v", err)
	}
	providerSvc, err := providers.NewService(providers.Config{
		Repo: database, Secrets: vaultSvc, CodexBin: "codex", CodexWorkDir: t.TempDir(),
		AccountsDir: t.TempDir(), Now: now, NewID: core.NewID,
	})
	if err != nil {
		t.Fatalf("providers service: %v", err)
	}
	if err := database.InsertProviderAccount(context.Background(), core.ProviderAccount{
		ID: testAccountID, OwnerID: testOwner, Kind: "anthropic", Label: "default", CreatedAt: now(), UpdatedAt: now(),
	}); err != nil {
		t.Fatalf("seed provider account: %v", err)
	}
	localToken, err := auth.NewLocalToken(testToken, vaultSvc)
	if err != nil {
		t.Fatalf("local token: %v", err)
	}
	hub := events.NewHub()
	convSvc, err := conversations.NewService(conversations.Config{
		Repo: database, Gateway: gw, Now: now, NewID: core.NewID, Events: hub,
	})
	if err != nil {
		t.Fatalf("conversations service: %v", err)
	}
	routinesSvc, err := routines.NewService(database, core.RunOnPC, time.UTC, noRunExecutor{}, nil, now, core.NewID)
	if err != nil {
		t.Fatalf("routines service: %v", err)
	}
	toolsSvc, err := tools.NewService(database, now, core.NewID)
	if err != nil {
		t.Fatalf("tools service: %v", err)
	}
	t.Cleanup(toolsSvc.Close)
	// Cleanups run last-in first-out: release blocked runs, then wait for them to end.
	t.Cleanup(convSvc.Wait)
	if releaseGateway != nil {
		t.Cleanup(releaseGateway)
	}
	h, err := NewHandler(Config{
		Agents:         agentSvc,
		Accounts:       providerSvc,
		Memories:       memorySvc,
		Approvals:      approvalSvc,
		Devices:        devicesSvc,
		Usage:          usageSvc,
		Vault:          vaultSvc,
		Audit:          database,
		Conversations:  convSvc,
		Events:         hub,
		Routines:       routinesSvc,
		Tools:          toolsSvc,
		Token:          localToken,
		OwnerID:        testOwner,
		DeviceID:       testDeviceID,
		TrustedProxies: trusted,
		CORSOrigins:    []string{testDesktopOrigin},
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	return h, database, approvalSvc
}

// noRunExecutor backs the routines service in API tests, which only queue occurrences.
type noRunExecutor struct{}

func (noRunExecutor) Execute(context.Context, core.Routine, core.Occurrence) (routines.Result, error) {
	return routines.Result{}, errors.New("routines: execution is not wired in api tests")
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
		fmt.Sprintf(`{"name":"  Calango  ","token_limit_month":100000,%s}`, testChain))
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
		fmt.Sprintf(`{"name":"Calango 2","token_limit_month":200000,"run_on":"vps",%s}`, testChain))
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

func TestApprovalDecisionAPIExecutesApprovedSnapshot(t *testing.T) {
	h, database, approvalSvc := newTestEnvWithApproval(t, nil)
	created := call(t, h, http.MethodPost, "/v1/agents", testToken,
		fmt.Sprintf(`{"name":"Approvals","token_limit_month":100000,%s}`, testChain))
	expectStatus(t, created, http.StatusCreated)
	var agent struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &agent); err != nil {
		t.Fatalf("decode agent: %v", err)
	}

	ctx := context.Background()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	conversationID, err := core.NewID(now)
	if err != nil {
		t.Fatalf("conversation id: %v", err)
	}
	if err := database.InsertConversation(ctx, core.Conversation{
		ID: conversationID, Kind: "dm", CreatedAt: now,
	}, agent.ID); err != nil {
		t.Fatalf("insert conversation: %v", err)
	}
	runID, err := core.NewID(now)
	if err != nil {
		t.Fatalf("run id: %v", err)
	}
	started := now
	if err := database.AdmitRun(ctx, nil, core.Run{
		ID: runID, AgentID: agent.ID, ConversationID: conversationID,
		IdempotencyKey: runID, Trigger: "message", Status: core.RunStatus("running"),
		StartedAt: &started,
	}); err != nil {
		t.Fatalf("insert run: %v", err)
	}

	action := core.Action{
		Kind: core.ActionConnectorWrite, Target: "notes/create",
		Args: json.RawMessage(`{"title":"groceries"}`),
	}
	executed := make(chan core.Action, 1)
	approvalSvc.SetExecutor(core.ActionConnectorWrite, func(_ context.Context, _, _ string, action core.Action) error {
		executed <- action
		return nil
	})
	pending, err := approvalSvc.Request(ctx, testOwner, agent.ID, runID, action)
	if err != nil {
		t.Fatalf("request approval: %v", err)
	}
	if pending.Status != core.ApprovalPending {
		t.Fatalf("request status = %s, want pending", pending.Status)
	}

	decided := call(t, h, http.MethodPost, "/v1/approvals/"+pending.ID+"/decide",
		testToken, `{"decision":"approve"}`)
	expectStatus(t, decided, http.StatusOK)
	var response approvalResponse
	if err := json.Unmarshal(decided.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode decision: %v", err)
	}
	if response.ID != pending.ID || response.Status != string(core.ApprovalApproved) {
		t.Fatalf("decision response = %+v, want approved %s", response, pending.ID)
	}

	select {
	case got := <-executed:
		if got.Kind != action.Kind || got.Target != action.Target || string(got.Args) != string(action.Args) {
			t.Fatalf("executor received %+v, want the approved snapshot %+v", got, action)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("approved action did not reach the executor")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		receipt, err := database.GetApproval(ctx, testOwner, pending.ID)
		if err != nil {
			t.Fatalf("get approval receipt: %v", err)
		}
		if receipt.Outcome == core.OutcomeCompleted {
			if receipt.DecidedByDeviceID != testDeviceID {
				t.Fatalf("decided by %q, want %q", receipt.DecidedByDeviceID, testDeviceID)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("approval outcome = %q, want completed", receipt.Outcome)
		}
		time.Sleep(time.Millisecond)
	}
}

// blockingGateway stands in for the model gateway. A call waits until its run is
// stopped or the test ends, so run states stay known without a real provider.
type blockingGateway struct {
	done chan struct{}
}

func (g *blockingGateway) Call(ctx context.Context, _ gateway.Caller, _ gateway.Request, _ func(gateway.Chunk)) (gateway.Usage, error) {
	select {
	case <-ctx.Done():
		return gateway.Usage{}, ctx.Err()
	case <-g.done:
		return gateway.Usage{}, errors.New("gateway released by test")
	}
}

type failingGateway struct {
	err error
}

func (g *failingGateway) Call(context.Context, gateway.Caller, gateway.Request, func(gateway.Chunk)) (gateway.Usage, error) {
	return gateway.Usage{}, g.err
}
