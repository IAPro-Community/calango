// Package mcpgw serves the MCP endpoint that a model turn uses to reach the
// connectors its agent was granted. Each turn gets its own bearer token bound to
// one owner, agent and run. Reads run directly; writes wait for an approval.
package mcpgw

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"calango/internal/approvals"
	"calango/internal/core"
	"calango/internal/gateway"
	"calango/internal/tools"
)

const (
	// Path is where the MCP endpoint is served on the daemon.
	Path = "/mcp"
	// DefaultWaitWindow bounds how long a write call waits for a decision.
	DefaultWaitWindow = 2 * time.Minute

	nameSeparator = "__"
	resultTTL     = time.Hour
	pollInterval  = 250 * time.Millisecond
)

// remoteNamePattern limits remote tool names to what the model can call.
var remoteNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// rememberTool is the built-in tool that saves a memory of the turn's agent.
const rememberTool = "memory_save"

type turnKey struct{}

// turn is the authority of one model run.
type turn struct {
	ownerID string
	agentID string
	runID   string
}

// stored is the upstream result of an approved write, kept until it is read or
// it is older than resultTTL. Results are in memory only.
type stored struct {
	result *mcp.CallToolResult
	at     time.Time
}

// RunStatuses moves a run between running and waiting_approval. The store implements it.
type RunStatuses interface {
	SetRunStatus(ctx context.Context, id string, from, to core.RunStatus) (bool, error)
}

// Memories saves what the agent remembers. The memory service implements it. Nil
// leaves memory_save out of the tool list.
type Memories interface {
	Create(ctx context.Context, ownerID string, m core.Memory) (core.Memory, error)
}

// Config wires the gateway. URL is the endpoint as the local model CLI reaches it.
type Config struct {
	Tools     *tools.Service
	Approvals *approvals.Service
	Runs      RunStatuses
	Memory    Memories
	URL       string
	// WaitWindow bounds the wait for a write decision. Zero uses DefaultWaitWindow.
	WaitWindow time.Duration
	Logger     *slog.Logger
}

// Gateway holds the live turns and serves the MCP endpoint for them.
type Gateway struct {
	cfg     Config
	window  time.Duration
	logger  *slog.Logger
	handler http.Handler

	mu      sync.Mutex
	turns   map[[sha256.Size]byte]turn
	results map[string]stored
}

// New builds the gateway. Its Handler must be mounted at Path outside the
// daemon's user-session auth: the turn token is the only credential there.
func New(cfg Config) (*Gateway, error) {
	if cfg.Tools == nil || cfg.Approvals == nil || cfg.Runs == nil {
		return nil, errors.New("mcp gateway needs a tools service, an approvals service and run statuses")
	}
	if cfg.URL == "" {
		return nil, errors.New("mcp gateway needs its endpoint URL")
	}
	g := &Gateway{
		cfg:     cfg,
		window:  cfg.WaitWindow,
		logger:  cfg.Logger,
		turns:   make(map[[sha256.Size]byte]turn),
		results: make(map[string]stored),
	}
	if g.window <= 0 {
		g.window = DefaultWaitWindow
	}
	if g.logger == nil {
		g.logger = slog.Default()
	}
	mcpHandler := mcp.NewStreamableHTTPHandler(g.serverFor, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
		Logger:       g.logger,
	})
	g.handler = g.authenticate(mcpHandler)
	return g, nil
}

// Handler serves the MCP endpoint.
func (g *Gateway) Handler() http.Handler { return g.handler }

// Open starts a turn for one run. It returns a nil endpoint when the agent has no
// grants and no built-in tools, so the model gets no tools. release must be called
// when the model call ends; it revokes the token.
func (g *Gateway) Open(ctx context.Context, ownerID, agentID, runID string) (*gateway.ToolEndpoint, func(), error) {
	grants, err := g.cfg.Tools.Grants(ctx, ownerID, agentID)
	if err != nil {
		return nil, nil, err
	}
	if len(grants) == 0 && g.cfg.Memory == nil {
		return nil, func() {}, nil
	}
	token, err := newToken()
	if err != nil {
		return nil, nil, err
	}
	key := digest(token)
	g.mu.Lock()
	g.turns[key] = turn{ownerID: ownerID, agentID: agentID, runID: runID}
	g.mu.Unlock()
	release := func() {
		g.mu.Lock()
		delete(g.turns, key)
		g.mu.Unlock()
	}
	return &gateway.ToolEndpoint{URL: g.cfg.URL, Token: token}, release, nil
}

// Execute runs an approved write. The snapshot binds connector name, immutable
// connector ID and tool. Old snapshots without identity must be approved again.
func (g *Gateway) Execute(ctx context.Context, ownerID, approvalID string, action core.Action) error {
	connName, rest, ok := strings.Cut(action.Target, "/")
	connID, tool, hasID := strings.Cut(rest, "/")
	if !ok || !hasID || !remoteNamePattern.MatchString(tool) {
		return errors.New("connector approval lacks a bound connector identity; request a new approval")
	}
	ap, err := g.cfg.Approvals.Get(ctx, ownerID, approvalID)
	if err != nil {
		return err
	}
	conn, err := g.cfg.Tools.Get(ctx, ownerID, connID)
	if err != nil || conn.Name != connName {
		return errors.New("approved connector no longer exists")
	}
	if err := g.checkGrant(ctx, ownerID, ap.AgentID, conn.ID, false); err != nil {
		return err
	}
	res, err := g.cfg.Tools.CallRemote(ctx, conn, tool, action.Args)
	if err != nil {
		return errors.New("approved connector call failed; outcome may be unknown, do not retry automatically")
	}
	if res.IsError {
		return errors.New("approved connector reported an error; do not retry automatically")
	}
	g.store(approvalID, res)
	return nil
}

// checkGrant revalidates current authority at dispatch, rather than relying on
// a grant captured during discovery. It cannot undo an already-started call.
func (g *Gateway) checkGrant(ctx context.Context, ownerID, agentID, connectorID string, readOnly bool) error {
	grants, err := g.cfg.Tools.Grants(ctx, ownerID, agentID)
	if err != nil {
		return errors.New("connector grant could not be verified")
	}
	for _, grant := range grants {
		if grant.ConnectorID == connectorID && (grant.Access == core.AccessWrite || (readOnly && grant.Access == core.AccessRead)) {
			return nil
		}
	}
	return errors.New("connector grant was revoked or does not authorize this operation")
}

// LoopbackURL builds the endpoint URL a local CLI uses for a daemon listening on addr.
func LoopbackURL(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("listen address %q: %w", addr, err)
	}
	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + Path, nil
}

// authenticate rejects requests whose bearer token is not a live turn.
func (g *Gateway) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		g.mu.Lock()
		t, ok := g.turns[digest(token)]
		g.mu.Unlock()
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), turnKey{}, t)))
	})
}

// serverFor builds the MCP server for one request. It is built per request so a
// grant change applies to the next request.
func (g *Gateway) serverFor(r *http.Request) *mcp.Server {
	t, ok := r.Context().Value(turnKey{}).(turn)
	if !ok {
		return nil
	}
	return g.buildServer(r.Context(), t)
}

// buildServer advertises the built-in tools and the tools the turn's grants allow.
// A connector that cannot list its tools is skipped with a log line, not a failed
// request.
func (g *Gateway) buildServer(ctx context.Context, t turn) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{Name: "calango", Version: "1"}, nil)
	if g.cfg.Memory != nil {
		srv.AddTool(&mcp.Tool{
			Name:        rememberTool,
			Description: "Save one fact, preference or decision about the person for future runs. The person can see and delete it.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{"type": "string", "description": "The memory, as one self-contained sentence."},
					"kind": map[string]any{"type": "string", "enum": []string{"fact", "preference", "decision"}},
				},
				"required": []string{"text"},
			},
		}, g.remember(t))
	}
	grants, err := g.cfg.Tools.Grants(ctx, t.ownerID, t.agentID)
	if err != nil {
		g.logger.Warn("mcp grants unavailable", "agent", t.agentID, "error", err)
		return srv
	}
	conns, err := g.cfg.Tools.List(ctx, t.ownerID)
	if err != nil {
		g.logger.Warn("mcp connectors unavailable", "error", err)
		return srv
	}
	byID := make(map[string]core.Connector, len(conns))
	for _, c := range conns {
		byID[c.ID] = c
	}
	for _, gr := range grants {
		conn, ok := byID[gr.ConnectorID]
		if !ok {
			continue
		}
		remote, err := g.cfg.Tools.RemoteTools(ctx, conn)
		if err != nil {
			g.logger.Warn("mcp connector tools unavailable", "connector", conn.Name, "error", err)
			continue
		}
		for _, rt := range remote {
			readOnly := rt.Annotations != nil && rt.Annotations.ReadOnlyHint
			if !remoteNamePattern.MatchString(rt.Name) {
				g.logger.Warn("mcp tool skipped: unsupported name", "connector", conn.Name, "tool", rt.Name)
				continue
			}
			if gr.Access == core.AccessRead && !readOnly {
				continue
			}
			if !isObjectSchema(rt.InputSchema) {
				g.logger.Warn("mcp tool skipped: input schema is not an object", "connector", conn.Name, "tool", rt.Name)
				continue
			}
			srv.AddTool(&mcp.Tool{
				Name:        conn.Name + nameSeparator + rt.Name,
				Description: rt.Description,
				InputSchema: rt.InputSchema,
				Annotations: rt.Annotations,
			}, g.handle(t, conn, rt.Name, readOnly))
		}
	}
	return srv
}

func (g *Gateway) handle(t turn, conn core.Connector, tool string, readOnly bool) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := g.checkGrant(ctx, t.ownerID, t.agentID, conn.ID, readOnly); err != nil {
			return errorResult(err.Error()), nil
		}
		if readOnly {
			res, err := g.cfg.Tools.CallRemote(ctx, conn, tool, req.Params.Arguments)
			if err != nil {
				return errorResult(err.Error()), nil
			}
			return res, nil
		}
		return g.callWithApproval(ctx, t, conn, tool, req.Params.Arguments), nil
	}
}

// remember saves a memory of the turn's agent. The scope and agent come from the
// turn, never from the arguments, so a model cannot write into another agent's memory.
// It needs no approval: the person sees and deletes every memory.
func (g *Gateway) remember(t turn) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var in struct {
			Text string `json:"text"`
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &in); err != nil {
			return errorResult("memory_save needs an object with text and an optional kind"), nil
		}
		m, err := g.cfg.Memory.Create(ctx, t.ownerID, core.Memory{
			Scope: core.ScopeAgent, AgentID: t.agentID, Kind: core.MemoryKind(in.Kind), Text: in.Text,
		})
		if err != nil {
			return errorResult("could not save memory: " + err.Error()), nil
		}
		return textResult("Saved memory " + m.ID + "."), nil
	}
}

// callWithApproval requests an approval for the exact arguments and waits for the
// decision, up to the wait window. The tool result is returned by the approval's
// executor, which stores it under the approval ID.
func (g *Gateway) callWithApproval(ctx context.Context, t turn, conn core.Connector, tool string, args json.RawMessage) *mcp.CallToolResult {
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	action := core.Action{Kind: core.ActionConnectorWrite, Target: conn.Name + "/" + conn.ID + "/" + tool, Args: args}
	ap, err := g.cfg.Approvals.Request(ctx, t.ownerID, t.agentID, t.runID, action)
	if err != nil {
		return errorResult("could not request approval: " + err.Error())
	}
	if ap.Status == core.ApprovalPending {
		// The run keeps its slot but waits for the person. It is running again once the
		// decision settles or this call stops waiting, even if the request was cancelled.
		bg := context.WithoutCancel(ctx)
		g.moveRun(bg, t.runID, core.RunRunning, core.RunWaitingApproval)
		defer g.moveRun(bg, t.runID, core.RunWaitingApproval, core.RunRunning)
	}
	return g.await(ctx, t.ownerID, ap.ID)
}

func (g *Gateway) await(ctx context.Context, ownerID, approvalID string) *mcp.CallToolResult {
	timer := time.NewTimer(g.window)
	defer timer.Stop()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		ap, err := g.cfg.Approvals.Get(ctx, ownerID, approvalID)
		if err != nil {
			return errorResult("could not read approval " + approvalID + ": " + err.Error())
		}
		if res, done := g.settled(ap); done {
			return res
		}
		select {
		case <-ctx.Done():
			return errorResult("the call was cancelled; approval " + approvalID + " is still recorded")
		case <-timer.C:
			return textResult("Approval " + approvalID + " is still pending. Do not repeat this call; the person has not decided yet.")
		case <-ticker.C:
		}
	}
}

// moveRun changes a run's status between running and waiting_approval. A run that
// is no longer in the expected state is logged, not fatal.
func (g *Gateway) moveRun(ctx context.Context, runID string, from, to core.RunStatus) {
	moved, err := g.cfg.Runs.SetRunStatus(ctx, runID, from, to)
	if err != nil {
		g.logger.Error("change run status", "run", runID, "to", to, "err", err)
		return
	}
	if !moved {
		g.logger.Warn("run was not in the expected state", "run", runID, "from", from, "to", to)
	}
}

// settled maps an approval to a tool result once it can no longer change.
func (g *Gateway) settled(ap core.Approval) (*mcp.CallToolResult, bool) {
	switch ap.Status {
	case core.ApprovalDenied:
		return textResult("The person denied this action. Do not retry it."), true
	case core.ApprovalExpired:
		return textResult("The approval expired without a decision. Do not retry unless the person asks."), true
	case core.ApprovalApproved:
		switch ap.Outcome {
		case "":
			return nil, false
		case core.OutcomeCompleted:
			if res, ok := g.take(ap.ID); ok {
				return res, true
			}
			return textResult("The action completed; its result was not kept (the daemon restarted or the result expired)."), true
		case core.OutcomeFailed:
			return errorResult("The action failed: " + ap.OutcomeReason), true
		default:
			return errorResult("The action did not run."), true
		}
	}
	return nil, false
}

func (g *Gateway) store(approvalID string, res *mcp.CallToolResult) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	for id, s := range g.results {
		if now.Sub(s.at) > resultTTL {
			delete(g.results, id)
		}
	}
	g.results[approvalID] = stored{result: res, at: now}
}

func (g *Gateway) take(approvalID string) (*mcp.CallToolResult, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	s, ok := g.results[approvalID]
	delete(g.results, approvalID)
	if !ok || time.Since(s.at) > resultTTL {
		return nil, false
	}
	return s.result, true
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("mcp token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func digest(token string) [sha256.Size]byte { return sha256.Sum256([]byte(token)) }

func isObjectSchema(schema any) bool {
	raw, err := json.Marshal(schema)
	if err != nil {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	return m["type"] == "object"
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func errorResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}
