// Package tools keeps the MCP connectors the owner has added and the grants that
// let an agent use them. It holds one live client session per connector.
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"calango/internal/core"
)

// connectTimeout bounds the MCP handshake only. A session outlives the request that
// opened it, so the handshake must not inherit that request's context.
const connectTimeout = 30 * time.Second

// Repository is the persistence the service needs. *store.Store implements it.
type Repository interface {
	GetAgent(ctx context.Context, ownerID, id string) (core.Agent, error)
	CreateConnector(ctx context.Context, c core.Connector) error
	ListConnectors(ctx context.Context, ownerID string) ([]core.Connector, error)
	GetConnector(ctx context.Context, ownerID, id string) (core.Connector, error)
	DeleteConnector(ctx context.Context, ownerID, id string) error
	SetAgentTool(ctx context.Context, t core.AgentTool) error
	RemoveAgentTool(ctx context.Context, agentID, connectorID string) error
	ListAgentTools(ctx context.Context, agentID string) ([]core.AgentTool, error)
}

// Service owns connectors, grants and the sessions to the servers behind them.
type Service struct {
	repo  Repository
	now   func() time.Time
	newID func(time.Time) (string, error)

	mu       sync.Mutex
	sessions map[string]*mcp.ClientSession
}

// NewService builds the service.
func NewService(repo Repository, now func() time.Time, newID func(time.Time) (string, error)) (*Service, error) {
	if repo == nil || now == nil || newID == nil {
		return nil, errors.New("tools service needs a repository, a clock and an id generator")
	}
	return &Service{repo: repo, now: now, newID: newID, sessions: make(map[string]*mcp.ClientSession)}, nil
}

// NewConnector is what a person sends to add a connector.
type NewConnector struct {
	Name      string
	Transport core.ConnectorTransport
	Command   string
	Args      []string
	URL       string
}

// Register validates and stores a connector. No session is opened yet.
func (s *Service) Register(ctx context.Context, ownerID string, in NewConnector) (core.Connector, error) {
	id, err := s.newID(s.now())
	if err != nil {
		return core.Connector{}, fmt.Errorf("connector id: %w", err)
	}
	now := s.now().UTC()
	c := core.Connector{
		ID: id, OwnerID: ownerID, Name: in.Name, Transport: in.Transport,
		Command: in.Command, Args: in.Args, URL: in.URL, CreatedAt: now, UpdatedAt: now,
	}
	if err := c.Validate(); err != nil {
		return core.Connector{}, err
	}
	if err := s.repo.CreateConnector(ctx, c); err != nil {
		return core.Connector{}, err
	}
	return c, nil
}

// List returns the owner's connectors.
func (s *Service) List(ctx context.Context, ownerID string) ([]core.Connector, error) {
	return s.repo.ListConnectors(ctx, ownerID)
}

// Get returns one of the owner's connectors.
func (s *Service) Get(ctx context.Context, ownerID, id string) (core.Connector, error) {
	return s.repo.GetConnector(ctx, ownerID, id)
}

// Delete removes a connector, its grants, and its live session.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	if err := s.repo.DeleteConnector(ctx, ownerID, id); err != nil {
		return err
	}
	s.drop(id)
	return nil
}

// Grant gives an agent access to a connector. Both must belong to the owner.
func (s *Service) Grant(ctx context.Context, ownerID, agentID, connectorID string, access core.ToolAccess) error {
	if !core.ValidAccess(access) {
		return fmt.Errorf("%w: access must be read or write; remove the grant to take it away", core.ErrInvalidInput)
	}
	if err := s.owns(ctx, ownerID, agentID, connectorID); err != nil {
		return err
	}
	return s.repo.SetAgentTool(ctx, core.AgentTool{AgentID: agentID, ConnectorID: connectorID, Access: access})
}

// Revoke removes an agent's grant on a connector.
func (s *Service) Revoke(ctx context.Context, ownerID, agentID, connectorID string) error {
	if err := s.owns(ctx, ownerID, agentID, connectorID); err != nil {
		return err
	}
	return s.repo.RemoveAgentTool(ctx, agentID, connectorID)
}

// Grants returns an agent's grants. The agent must belong to the owner.
func (s *Service) Grants(ctx context.Context, ownerID, agentID string) ([]core.AgentTool, error) {
	if _, err := s.repo.GetAgent(ctx, ownerID, agentID); err != nil {
		return nil, err
	}
	return s.repo.ListAgentTools(ctx, agentID)
}

// ErrUnavailable means a connector could not be reached or did not list its tools.
// Its message is fixed: the underlying error can name a command or a URL.
var ErrUnavailable = errors.New("tools: connector unavailable")

// ToolInfo describes one tool a connector advertises.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReadOnly    bool   `json:"read_only"`
}

// Catalog lists the tools of a connector the owner registered.
func (s *Service) Catalog(ctx context.Context, ownerID, id string) ([]ToolInfo, error) {
	c, err := s.Get(ctx, ownerID, id)
	if err != nil {
		return nil, err
	}
	remote, err := s.RemoteTools(ctx, c)
	if err != nil {
		return nil, err
	}
	out := make([]ToolInfo, 0, len(remote))
	for _, t := range remote {
		out = append(out, ToolInfo{
			Name:        t.Name,
			Description: t.Description,
			ReadOnly:    t.Annotations != nil && t.Annotations.ReadOnlyHint,
		})
	}
	return out, nil
}

// RemoteTools lists the tools one connector advertises.
func (s *Service) RemoteTools(ctx context.Context, c core.Connector) ([]*mcp.Tool, error) {
	sess, err := s.session(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	var out []*mcp.Tool
	for t, err := range sess.Tools(ctx, nil) {
		if err != nil {
			s.drop(c.ID)
			return nil, fmt.Errorf("%w: list tools: %v", ErrUnavailable, err)
		}
		out = append(out, t)
	}
	return out, nil
}

// CallRemote calls one tool on a connector. A protocol failure drops the session so
// the next call reconnects. A tool that reports its own error comes back in the
// result with IsError set, not as an error.
func (s *Service) CallRemote(ctx context.Context, c core.Connector, tool string, args json.RawMessage) (*mcp.CallToolResult, error) {
	sess, err := s.session(ctx, c)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		s.drop(c.ID)
		return nil, fmt.Errorf("call %s/%s: %w", c.Name, tool, err)
	}
	return res, nil
}

// Close ends every live session. Call it when the daemon stops.
func (s *Service) Close() {
	s.mu.Lock()
	sessions := s.sessions
	s.sessions = make(map[string]*mcp.ClientSession)
	s.mu.Unlock()
	for _, sess := range sessions {
		_ = sess.Close()
	}
}

func (s *Service) owns(ctx context.Context, ownerID, agentID, connectorID string) error {
	if _, err := s.repo.GetAgent(ctx, ownerID, agentID); err != nil {
		return err
	}
	_, err := s.repo.GetConnector(ctx, ownerID, connectorID)
	return err
}

// session returns the live session for a connector, connecting it on first use.
// Two callers may race to connect; the loser closes its session.
func (s *Service) session(ctx context.Context, c core.Connector) (*mcp.ClientSession, error) {
	s.mu.Lock()
	if sess, ok := s.sessions[c.ID]; ok {
		s.mu.Unlock()
		return sess, nil
	}
	s.mu.Unlock()

	transport, err := transportFor(c)
	if err != nil {
		return nil, err
	}
	connectCtx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "calango", Version: "1"}, nil)
	sess, err := client.Connect(connectCtx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", c.Name, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.sessions[c.ID]; ok {
		_ = sess.Close()
		return existing, nil
	}
	s.sessions[c.ID] = sess
	return sess, nil
}

func (s *Service) drop(connectorID string) {
	s.mu.Lock()
	sess, ok := s.sessions[connectorID]
	delete(s.sessions, connectorID)
	s.mu.Unlock()
	if ok {
		_ = sess.Close()
	}
}

// transportFor builds the MCP transport. A stdio server gets only PATH and HOME from
// the daemon: the daemon environment holds the vault key, which no connector needs.
func transportFor(c core.Connector) (mcp.Transport, error) {
	switch c.Transport {
	case core.TransportStdio:
		cmd := exec.Command(c.Command, c.Args...)
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME")}
		return &mcp.CommandTransport{Command: cmd}, nil
	case core.TransportHTTP:
		return &mcp.StreamableClientTransport{Endpoint: c.URL}, nil
	}
	return nil, fmt.Errorf("%w: unknown transport %q", core.ErrInvalidInput, c.Transport)
}
