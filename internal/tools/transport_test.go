package tools

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"calango/internal/core"
)

// A stdio server must not inherit the daemon environment: it holds the vault key.
func TestStdioServerGetsOnlyPathAndHome(t *testing.T) {
	t.Setenv("CALANGO_VAULT_KEY", "must-not-leak")
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("HOME", "/home/test")

	tr, err := transportFor(core.Connector{Name: "notes", Transport: core.TransportStdio, Command: "mcp-notes"})
	if err != nil {
		t.Fatalf("transport: %v", err)
	}
	ct, ok := tr.(*mcp.CommandTransport)
	if !ok {
		t.Fatalf("transport is %T, want *mcp.CommandTransport", tr)
	}
	env := strings.Join(ct.Command.Env, "\n")
	if env != "PATH=/usr/bin\nHOME=/home/test" {
		t.Fatalf("stdio env = %q, want only PATH and HOME", env)
	}
}
