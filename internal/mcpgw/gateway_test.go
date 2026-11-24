package mcpgw_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"calango/internal/agents"
	"calango/internal/approvals"
	"calango/internal/core"
	"calango/internal/mcpgw"
	"calango/internal/memory"
	"calango/internal/store"
	"calango/internal/tools"
)

const owner = "owner"

// remoteServer is the connector behind the gateway: one read tool and one write tool.
// Each tool echoes its arguments so tests can see exactly what reached it.
func remoteServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := mcp.NewServer(&mcp.Implementation{Name: "notes", Version: "1"}, nil)
	schema := map[string]any{"type": "object"}
	srv.AddTool(&mcp.Tool{
		Name: "read_note", Description: "reads a note", InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return textResult("read " + string(req.Params.Arguments)), nil
	})
	srv.AddTool(&mcp.Tool{Name: "create_note", Description: "writes a note", InputSchema: schema},
		func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return textResult("created " + string(req.Params.Arguments)), nil
		})
	ts := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil))
	t.Cleanup(ts.Close)
	return ts
}

func textResult(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

type env struct {
	st      *store.Store
	apps    *approvals.Service
	tools   *tools.Service
	mem     *memory.Service
	agentID string
	runID   string
}

// newEnv builds a store with one agent and one running run. Cleanups run in the
// reverse order of registration, so the remote server must be created before it.
func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	st, err := store.OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	clk := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return clk }

	apps, err := approvals.NewService(st, approvals.DefaultTTL, now, core.NewID)
	if err != nil {
		t.Fatalf("approvals: %v", err)
	}
	tsvc, err := tools.NewService(st, now, core.NewID)
	if err != nil {
		t.Fatalf("tools: %v", err)
	}
	t.Cleanup(tsvc.Close)
	mem, err := memory.NewService(st, now, core.NewID)
	if err != nil {
		t.Fatalf("memory: %v", err)
	}

	ag, err := agents.NewService(st, now, core.NewID)
	if err != nil {
		t.Fatalf("agents: %v", err)
	}
	accountID, _ := core.NewID(clk)
	if err := st.InsertProviderAccount(ctx, core.ProviderAccount{ID: accountID, OwnerID: owner, Kind: "codex", Label: "default", CreatedAt: clk, UpdatedAt: clk}); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	view, err := ag.Create(ctx, owner, core.Agent{
		Name: "Calango", TokenLimitMonth: 1000,
		Providers: []core.ProviderRef{{AccountID: accountID, Kind: "codex", Label: "default", Model: "m"}},
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	conv, _ := core.NewID(clk)
	if err := st.InsertConversation(ctx, core.Conversation{ID: conv, Kind: "dm", CreatedAt: clk}, view.ID); err != nil {
		t.Fatalf("insert conversation: %v", err)
	}
	runID, _ := core.NewID(clk)
	started := clk
	if err := st.AdmitRun(ctx, nil, core.Run{
		ID: runID, AgentID: view.ID, ConversationID: conv, IdempotencyKey: runID,
		Trigger: "message", Status: core.RunStatus("running"), StartedAt: &started,
	}); err != nil {
		t.Fatalf("admit run: %v", err)
	}
	return &env{st: st, apps: apps, tools: tsvc, mem: mem, agentID: view.ID, runID: runID}
}

// registerNotes adds the remote connector and grants the agent the given access.
func (e *env) registerNotes(t *testing.T, remoteURL string, access core.ToolAccess) {
	t.Helper()
	ctx := context.Background()
	c, err := e.tools.Register(ctx, owner, tools.NewConnector{Name: "notes", Transport: core.TransportHTTP, URL: remoteURL})
	if err != nil {
		t.Fatalf("register connector: %v", err)
	}
	if err := e.tools.Grant(ctx, owner, e.agentID, c.ID, access); err != nil {
		t.Fatalf("grant: %v", err)
	}
}

// startGateway serves the gateway on a loopback listener. Its URL is known before
// the handler is wired, so the gateway can be configured with it.
func startGateway(t *testing.T, e *env, window time.Duration) *mcpgw.Gateway {
	t.Helper()
	ts := httptest.NewUnstartedServer(nil)
	endpoint := "http://" + ts.Listener.Addr().String() + mcpgw.Path
	gw, err := mcpgw.New(mcpgw.Config{Tools: e.tools, Approvals: e.apps, Runs: e.st, Memory: e.mem, URL: endpoint, WaitWindow: window})
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	e.apps.SetExecutor(core.ActionConnectorWrite, gw.Execute)
	ts.Config.Handler = gw.Handler()
	ts.Start()
	t.Cleanup(ts.Close)
	return gw
}

type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := r.Clone(r.Context())
	r2.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r2)
}

// connect opens an MCP client session as the model CLI would, with the turn token.
func connect(t *testing.T, endpoint, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "model-cli", Version: "1"}, nil)
	transport := &mcp.StreamableClientTransport{Endpoint: endpoint, HTTPClient: &http.Client{Transport: bearer{token}}}
	sess, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

func toolNames(t *testing.T, sess *mcp.ClientSession) map[string]bool {
	t.Helper()
	res, err := sess.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}
	return names
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("empty tool result")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content is %T, want text", res.Content[0])
	}
	return tc.Text
}

// waitPending returns the one pending approval, polling until it appears.
func (e *env) waitPending(t *testing.T) core.Approval {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		list, err := e.apps.List(context.Background(), owner, core.ApprovalPending)
		if err != nil {
			t.Fatalf("list approvals: %v", err)
		}
		if len(list) == 1 {
			return list[0]
		}
		if time.Now().After(deadline) {
			t.Fatalf("pending approvals = %d, want 1", len(list))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// waitRunStatus polls until the run has the wanted status, failing after 5 seconds.
func (e *env) waitRunStatus(t *testing.T, want core.RunStatus) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		run, err := e.st.GetRun(context.Background(), e.runID)
		if err != nil {
			t.Fatalf("read run: %v", err)
		}
		if run.Status == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("run status = %s, want %s", run.Status, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestReadGrantAdvertisesAndRunsOnlyReadTools(t *testing.T) {
	ctx := context.Background()
	remote := remoteServer(t)
	e := newEnv(t)
	e.registerNotes(t, remote.URL, core.AccessRead)
	gw := startGateway(t, e, 0)

	ep, release, err := gw.Open(ctx, owner, e.agentID, e.runID)
	if err != nil || ep == nil {
		t.Fatalf("open turn: ep=%v err=%v", ep, err)
	}
	t.Cleanup(release)
	sess := connect(t, ep.URL, ep.Token)

	if names := toolNames(t, sess); len(names) != 2 || !names["notes__read_note"] || !names["memory_save"] {
		t.Fatalf("read grant advertised %v, want notes__read_note and memory_save", names)
	}
	res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "notes__read_note", Arguments: map[string]any{"id": "n1"}})
	if err != nil || res.IsError {
		t.Fatalf("read call: res=%+v err=%v", res, err)
	}
	if got := resultText(t, res); got != `read {"id":"n1"}` {
		t.Fatalf("read result = %q", got)
	}
	// A hidden write tool is not callable even when the name is guessed.
	res, err = sess.CallTool(ctx, &mcp.CallToolParams{Name: "notes__create_note", Arguments: map[string]any{}})
	if err == nil && !res.IsError {
		t.Fatal("write tool ran under a read grant")
	}
}

func TestWriteGrantRoutesWritesThroughApproval(t *testing.T) {
	ctx := context.Background()
	remote := remoteServer(t)
	e := newEnv(t)
	e.registerNotes(t, remote.URL, core.AccessWrite)
	gw := startGateway(t, e, 10*time.Second)

	ep, release, err := gw.Open(ctx, owner, e.agentID, e.runID)
	if err != nil || ep == nil {
		t.Fatalf("open turn: ep=%v err=%v", ep, err)
	}
	t.Cleanup(release)
	sess := connect(t, ep.URL, ep.Token)
	if names := toolNames(t, sess); !names["notes__read_note"] || !names["notes__create_note"] {
		t.Fatalf("write grant advertised %v, want both tools", names)
	}

	type outcome struct {
		res *mcp.CallToolResult
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "notes__create_note", Arguments: map[string]any{"title": "milk"}})
		done <- outcome{res, err}
	}()
	ap := e.waitPending(t)
	if ap.Action.Target != "notes/create_note" {
		t.Fatalf("approval target = %q", ap.Action.Target)
	}
	if _, err := e.apps.Decide(ctx, owner, ap.ID, "dev-1", true, false); err != nil {
		t.Fatalf("approve: %v", err)
	}
	got := <-done
	if got.err != nil || got.res.IsError {
		t.Fatalf("approved write: res=%+v err=%v", got.res, got.err)
	}
	if text := resultText(t, got.res); text != `created {"title":"milk"}` {
		t.Fatalf("approved write result = %q; the connector must get the approved arguments", text)
	}
}

func TestWriteHoldsRunWhileWaitingForDecision(t *testing.T) {
	ctx := context.Background()
	remote := remoteServer(t)
	e := newEnv(t)
	e.registerNotes(t, remote.URL, core.AccessWrite)
	gw := startGateway(t, e, 10*time.Second)

	ep, release, err := gw.Open(ctx, owner, e.agentID, e.runID)
	if err != nil || ep == nil {
		t.Fatalf("open turn: ep=%v err=%v", ep, err)
	}
	t.Cleanup(release)
	sess := connect(t, ep.URL, ep.Token)

	done := make(chan *mcp.CallToolResult, 1)
	go func() {
		res, _ := sess.CallTool(ctx, &mcp.CallToolParams{Name: "notes__create_note", Arguments: map[string]any{"title": "eggs"}})
		done <- res
	}()
	ap := e.waitPending(t)
	e.waitRunStatus(t, core.RunWaitingApproval)
	if _, err := e.apps.Decide(ctx, owner, ap.ID, "dev-1", true, false); err != nil {
		t.Fatalf("approve: %v", err)
	}
	res := <-done
	if res == nil || res.IsError {
		t.Fatalf("approved write result = %+v", res)
	}
	e.waitRunStatus(t, core.RunRunning)
}

func TestDeniedWriteReturnsRefusalAndDoesNotRun(t *testing.T) {
	ctx := context.Background()
	remote := remoteServer(t)
	e := newEnv(t)
	e.registerNotes(t, remote.URL, core.AccessWrite)
	gw := startGateway(t, e, 10*time.Second)

	ep, release, err := gw.Open(ctx, owner, e.agentID, e.runID)
	if err != nil || ep == nil {
		t.Fatalf("open turn: ep=%v err=%v", ep, err)
	}
	t.Cleanup(release)
	sess := connect(t, ep.URL, ep.Token)

	done := make(chan *mcp.CallToolResult, 1)
	go func() {
		res, _ := sess.CallTool(ctx, &mcp.CallToolParams{Name: "notes__create_note", Arguments: map[string]any{"title": "x"}})
		done <- res
	}()
	ap := e.waitPending(t)
	if _, err := e.apps.Decide(ctx, owner, ap.ID, "dev-1", false, false); err != nil {
		t.Fatalf("deny: %v", err)
	}
	res := <-done
	if res == nil || !strings.Contains(resultText(t, res), "denied") {
		t.Fatalf("denied write result = %+v", res)
	}
	stored, err := e.apps.Get(ctx, owner, ap.ID)
	if err != nil || stored.Status != core.ApprovalDenied || stored.Outcome != core.OutcomeNotRun {
		t.Fatalf("denied approval = %+v, %v; want denied with outcome not_run", stored, err)
	}
}

func TestWriteWithoutDecisionReturnsPendingAndKeepsTheApproval(t *testing.T) {
	ctx := context.Background()
	remote := remoteServer(t)
	e := newEnv(t)
	e.registerNotes(t, remote.URL, core.AccessWrite)
	gw := startGateway(t, e, 300*time.Millisecond)

	ep, release, err := gw.Open(ctx, owner, e.agentID, e.runID)
	if err != nil || ep == nil {
		t.Fatalf("open turn: ep=%v err=%v", ep, err)
	}
	t.Cleanup(release)
	sess := connect(t, ep.URL, ep.Token)

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "notes__create_note", Arguments: map[string]any{"title": "y"}})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.Contains(resultText(t, res), "still pending") {
		t.Fatalf("timeout result = %q", resultText(t, res))
	}
	// The call stopped waiting, so the run is running again, not waiting_approval.
	run, err := e.st.GetRun(ctx, e.runID)
	if err != nil || run.Status != core.RunRunning {
		t.Fatalf("run after timeout = %s, %v; want running", run.Status, err)
	}
	e.waitPending(t)
}

func TestOpenWithoutGrantsOffersOnlyMemorySave(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	gw := startGateway(t, e, 0)
	ep, release, err := gw.Open(ctx, owner, e.agentID, e.runID)
	if err != nil || ep == nil {
		t.Fatalf("open turn: ep=%v err=%v", ep, err)
	}
	defer release()
	names := toolNames(t, connect(t, ep.URL, ep.Token))
	if len(names) != 1 || !names["memory_save"] {
		t.Fatalf("tools = %v, want only memory_save", names)
	}
}

func TestMemorySaveWritesAgentMemoryOnly(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t)
	gw := startGateway(t, e, 0)
	ep, release, err := gw.Open(ctx, owner, e.agentID, e.runID)
	if err != nil || ep == nil {
		t.Fatalf("open turn: ep=%v err=%v", ep, err)
	}
	defer release()
	sess := connect(t, ep.URL, ep.Token)

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "memory_save", Arguments: map[string]any{"text": "  Prefers short replies  "}})
	if err != nil || res.IsError {
		t.Fatalf("save memory: res=%+v err=%v", res, err)
	}
	list, err := e.mem.List(ctx, owner, core.ScopeAgent, e.agentID)
	if err != nil || len(list) != 1 {
		t.Fatalf("agent memories = %d, %v; want 1", len(list), err)
	}
	if list[0].Text != "Prefers short replies" || list[0].Kind != core.KindFact {
		t.Fatalf("memory = %+v; want trimmed text with kind fact", list[0])
	}

	empty, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "memory_save", Arguments: map[string]any{"text": "   "}})
	if err != nil {
		t.Fatalf("call with empty text: %v", err)
	}
	if !empty.IsError {
		t.Fatal("empty memory was accepted")
	}
	if list, _ := e.mem.List(ctx, owner, core.ScopeAgent, e.agentID); len(list) != 1 {
		t.Fatalf("agent memories after rejected save = %d, want 1", len(list))
	}
}

func TestEndpointRejectsMissingAndReleasedTokens(t *testing.T) {
	ctx := context.Background()
	remote := remoteServer(t)
	e := newEnv(t)
	e.registerNotes(t, remote.URL, core.AccessRead)
	gw := startGateway(t, e, 0)

	ep, release, err := gw.Open(ctx, owner, e.agentID, e.runID)
	if err != nil || ep == nil {
		t.Fatalf("open turn: ep=%v err=%v", ep, err)
	}
	post := func(token string) int {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("post: %v", err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}
	if code := post(""); code != http.StatusUnauthorized {
		t.Fatalf("no token: status %d, want 401", code)
	}
	if code := post("not-a-turn-token"); code != http.StatusUnauthorized {
		t.Fatalf("wrong token: status %d, want 401", code)
	}
	if code := post(ep.Token); code == http.StatusUnauthorized {
		t.Fatal("live turn token was rejected")
	}
	release()
	if code := post(ep.Token); code != http.StatusUnauthorized {
		t.Fatalf("released token: status %d, want 401", code)
	}
}

func TestLoopbackURL(t *testing.T) {
	cases := map[string]string{
		":7317":          "http://127.0.0.1:7317/mcp",
		"0.0.0.0:7317":   "http://127.0.0.1:7317/mcp",
		"[::]:7317":      "http://127.0.0.1:7317/mcp",
		"192.168.1.5:80": "http://192.168.1.5:80/mcp",
	}
	for addr, want := range cases {
		got, err := mcpgw.LoopbackURL(addr)
		if err != nil || got != want {
			t.Fatalf("LoopbackURL(%q) = %q, %v; want %q", addr, got, err, want)
		}
	}
	if _, err := mcpgw.LoopbackURL("no-port"); err == nil {
		t.Fatal("address without a port was accepted")
	}
}
