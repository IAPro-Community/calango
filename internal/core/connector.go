package core

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ConnectorTransport says how the daemon reaches an MCP server.
type ConnectorTransport string

const (
	TransportStdio ConnectorTransport = "stdio"
	TransportHTTP  ConnectorTransport = "http"
)

// ToolAccess is what an agent may do through one connector. An agent with no grant
// has no access; "none" is never stored.
type ToolAccess string

const (
	AccessRead  ToolAccess = "read"
	AccessWrite ToolAccess = "write"
)

// Connector is one MCP server the owner has connected. Name is the prefix of every
// tool it exposes to agents, so it is restricted to a safe identifier.
type Connector struct {
	ID        string
	OwnerID   string
	Name      string
	Transport ConnectorTransport
	Command   string
	Args      []string
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AgentTool is one grant: an agent's access to one connector.
type AgentTool struct {
	AgentID     string
	ConnectorID string
	Access      ToolAccess
}

var connectorNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,23}$`)

// Validate checks the fields a person can set. Stdio commands run on the node, so
// the command must be a plain path or name with no newline.
func (c Connector) Validate() error {
	if !connectorNamePattern.MatchString(c.Name) {
		return fmt.Errorf("%w: connector name must be 1-24 of a-z, 0-9, '-', starting with a letter or digit", ErrInvalidInput)
	}
	switch c.Transport {
	case TransportStdio:
		if strings.TrimSpace(c.Command) == "" || strings.ContainsAny(c.Command, "\n\r") {
			return fmt.Errorf("%w: stdio connector needs a command", ErrInvalidInput)
		}
		if c.URL != "" {
			return fmt.Errorf("%w: stdio connector takes no url", ErrInvalidInput)
		}
	case TransportHTTP:
		u, err := url.Parse(c.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("%w: http connector needs an http or https url", ErrInvalidInput)
		}
		if c.Command != "" || len(c.Args) > 0 {
			return fmt.Errorf("%w: http connector takes no command", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: transport must be stdio or http", ErrInvalidInput)
	}
	return nil
}

// ValidAccess reports whether a is a grant that can be stored.
func ValidAccess(a ToolAccess) bool {
	return a == AccessRead || a == AccessWrite
}
