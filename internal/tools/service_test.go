package tools_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"calango/internal/agents"
	"calango/internal/core"
	"calango/internal/store"
	"calango/internal/tools"
)

const owner = "owner"

// remoteServer is a small MCP server with one read-only tool and one write tool.
func remoteServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := mcp.NewServer(&mcp.Implementation{Name: "notes", Version: "1"}, nil)
	schema := map[string]any{"type": "object"}
	srv.AddTool(&mcp.Tool{
		Name: "read_note", Description: "reads a note", InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "read " + string(req.Params.Arguments)}}}, nil
	})
	srv.AddTool(&mcp.Tool{Name: "create_note", Description: "writes a note", InputSchema: schema},
		func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "created " + string(req.Params.Arguments)}}}, nil
		})
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, nil)
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts
}

type fixture struct {
	svc     *tools.Service
	st      *store.Store
	agentID string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	ctx := context.Background()
	st, err := store.OpenMemory(ctx)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	clk := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return clk }

	svc, err := tools.NewService(st, now, core.NewID)
	if err != nil {
		t.Fatalf("tools service: %v", err)
	}
	t.Cleanup(svc.Close)

	accountID, err := core.NewID(clk)
	if err != nil {
		t.Fatalf("account id: %v", err)
	}
	if err := st.InsertProviderAccount(ctx, core.ProviderAccount{ID: accountID, OwnerID: owner, Kind: "codex", Label: "default", CreatedAt: clk, UpdatedAt: clk}); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	ag, err := agents.NewService(st, now, core.NewID)
	if err != nil {
		t.Fatalf("agents service: %v", err)
	}
	view, err := ag.Create(ctx, owner, core.Agent{
		Name: "Calango", TokenLimitMonth: 1000,
		Providers: []core.ProviderRef{{AccountID: accountID, Kind: "codex", Label: "default", Model: "m"}},
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	return fixture{svc: svc, st: st, agentID: view.ID}
}

func TestRemoteToolsAndCallsGoThroughTheConnector(t *testing.T) {
	ctx := context.Background()
	ts := remoteServer(t)
	f := newFixture(t)

	c, err := f.svc.Register(ctx, owner, tools.NewConnector{Name: "notes", Transport: core.TransportHTTP, URL: ts.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	list, err := f.svc.RemoteTools(ctx, c)
	if err != nil {
		t.Fatalf("remote tools: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range list {
		names[tool.Name] = true
		if tool.Name == "read_note" && (tool.Annotations == nil || !tool.Annotations.ReadOnlyHint) {
			t.Fatal("read_note lost its readOnlyHint annotation")
		}
	}
	if !names["read_note"] || !names["create_note"] {
		t.Fatalf("remote tools = %v, want read_note and create_note", names)
	}

	res, err := f.svc.CallRemote(ctx, c, "read_note", json.RawMessage(`{"id":"n1"}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if got := res.Content[0].(*mcp.TextContent).Text; got != `read {"id":"n1"}` {
		t.Fatalf("call result = %q", got)
	}

	if _, err := f.svc.CallRemote(ctx, c, "no_such_tool", nil); err == nil {
		t.Fatal("calling an unknown tool succeeded")
	}
	res, err = f.svc.CallRemote(ctx, c, "read_note", nil)
	if err != nil {
		t.Fatalf("call after a failed call did not reconnect: %v", err)
	}
	if !strings.HasPrefix(res.Content[0].(*mcp.TextContent).Text, "read ") {
		t.Fatalf("reconnected call result = %+v", res.Content)
	}
}

func TestGrantsAreOwnerScopedAndValidated(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	c, err := f.svc.Register(ctx, owner, tools.NewConnector{Name: "notes", Transport: core.TransportHTTP, URL: "https://example.com/mcp"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := f.svc.Grant(ctx, owner, f.agentID, c.ID, core.ToolAccess("none")); !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("grant none err = %v, want invalid input", err)
	}
	if err := f.svc.Grant(ctx, "intruder", f.agentID, c.ID, core.AccessRead); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("other owner granted an agent: %v", err)
	}
	if _, err := f.svc.Get(ctx, "intruder", c.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("other owner read a connector: %v", err)
	}

	if err := f.svc.Grant(ctx, owner, f.agentID, c.ID, core.AccessRead); err != nil {
		t.Fatalf("grant read: %v", err)
	}
	if err := f.svc.Grant(ctx, owner, f.agentID, c.ID, core.AccessWrite); err != nil {
		t.Fatalf("upgrade to write: %v", err)
	}
	grants, err := f.svc.Grants(ctx, owner, f.agentID)
	if err != nil || len(grants) != 1 || grants[0].Access != core.AccessWrite {
		t.Fatalf("grants = %+v, %v; want one write grant", grants, err)
	}
	if err := f.svc.Revoke(ctx, owner, f.agentID, c.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if grants, _ := f.svc.Grants(ctx, owner, f.agentID); len(grants) != 0 {
		t.Fatalf("revoke kept grants: %+v", grants)
	}
}

func TestDeletingAConnectorRemovesItsGrants(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	c, err := f.svc.Register(ctx, owner, tools.NewConnector{Name: "notes", Transport: core.TransportHTTP, URL: "https://example.com/mcp"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := f.svc.Grant(ctx, owner, f.agentID, c.ID, core.AccessRead); err != nil {
		t.Fatalf("grant: %v", err)
	}
	if err := f.svc.Delete(ctx, owner, c.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if grants, _ := f.svc.Grants(ctx, owner, f.agentID); len(grants) != 0 {
		t.Fatalf("grants survived connector delete: %+v", grants)
	}
}

func TestRegisterRejectsAnUnsafeStdioCommand(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Register(context.Background(), owner, tools.NewConnector{
		Name: "shell", Transport: core.TransportStdio, Command: "sh\n-c",
	})
	if !errors.Is(err, core.ErrInvalidInput) {
		t.Fatalf("multi-line command err = %v, want invalid input", err)
	}
}
