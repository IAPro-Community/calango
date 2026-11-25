// Package harness runs model turns through agent CLIs. A CLI is a model
// provider behind the gateway: Calango builds the prompt, the gateway checks
// and records the budget, and the CLI runs the turn in an isolated process.
package harness

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"calango/internal/core"
	"calango/internal/gateway"
)

// ErrSignedOut means the CLI has no valid sign-in on this machine. The fix is
// to sign in again with the CLI, outside Calango.
var ErrSignedOut = errors.New("harness: CLI is signed out; sign in again with the CLI on this machine")

// ErrNotInstalled means the CLI binary was not found on PATH.
var ErrNotInstalled = errors.New("harness: CLI is not installed or not on PATH")

// CodexName is the gateway provider name of the Codex CLI harness.
const CodexName = core.ProviderKindCodex

// Status says whether a CLI can run a turn right now.
type Status struct {
	Installed bool
	SignedIn  bool
}

// Codex runs turns through the OpenAI Codex CLI (`codex exec`). It implements
// gateway.Provider. Runs are isolated from the person's Codex configuration
// and session history. Native CLI commands are not mediated by Calango.
type Codex struct {
	bin     string
	workDir string
	// home is the CODEX_HOME of this account. Empty means the person's default
	// Codex home, which is ADR 0013's shared default.
	home string
}

// NewCodex returns a Codex harness. bin is the CLI binary ("codex" when empty).
// workDir is the Codex working root for every run; it must already exist. home
// is the CODEX_HOME for this account; empty uses the default Codex home.
func NewCodex(bin, workDir, home string) (*Codex, error) {
	if workDir == "" {
		return nil, errors.New("harness: codex working directory is required")
	}
	if bin == "" {
		bin = "codex"
	}
	return &Codex{bin: bin, workDir: workDir, home: home}, nil
}

func (c *Codex) commandWithHome(ctx context.Context, home string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, c.bin, args...)
	cmd.Env = safeCLIEnv(os.Environ())
	if home != "" {
		// The explicitly selected account overrides the default home.
		cmd.Env = append(cmd.Env, "CODEX_HOME="+home)
	}
	return cmd
}

// Name implements gateway.Provider.
func (c *Codex) Name() string { return CodexName }

// MayHaveSideEffects includes native commands even without an MCP endpoint.
func (c *Codex) MayHaveSideEffects() bool { return true }

// Available reports whether the CLI is installed and signed in. It runs
// `codex login status`, which does not contact the model.
func (c *Codex) Available(ctx context.Context) (Status, error) {
	return c.available(ctx, c.home)
}

func (c *Codex) available(ctx context.Context, home string) (Status, error) {
	if _, err := exec.LookPath(c.bin); err != nil {
		return Status{}, nil
	}
	out, err := c.commandWithHome(ctx, home, "login", "status").CombinedOutput()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return Status{Installed: true}, fmt.Errorf("harness: codex login status: %w", ctxErr)
	}
	// A non-zero exit or missing "Logged in" text means signed out.
	signedIn := err == nil && strings.Contains(string(out), "Logged in")
	return Status{Installed: true, SignedIn: signedIn}, nil
}

// Complete implements gateway.Provider. It sends the prompt on stdin, waits
// for the CLI to finish, and streams the final message as one chunk. Codex
// output is not parsed as events, so tokens are always estimated.
func (c *Codex) Complete(ctx context.Context, req gateway.Request, stream func(gateway.Chunk)) (gateway.Usage, error) {
	home := c.home
	if req.CodexHome != "" {
		home = req.CodexHome
	}
	status, err := c.available(ctx, home)
	if err != nil {
		return gateway.Usage{}, err
	}
	if !status.Installed {
		return gateway.Usage{}, fmt.Errorf("%w: %s", ErrNotInstalled, c.bin)
	}
	if !status.SignedIn {
		return gateway.Usage{}, ErrSignedOut
	}

	outFile, err := os.CreateTemp("", "calango-codex-*.txt")
	if err != nil {
		return gateway.Usage{}, fmt.Errorf("harness: create output file: %w", err)
	}
	outPath := outFile.Name()
	if err := outFile.Close(); err != nil {
		_ = os.Remove(outPath)
		return gateway.Usage{}, fmt.Errorf("harness: close output file: %w", err)
	}
	defer func() { _ = os.Remove(outPath) }()

	// mcpTokenEnv names the variable that carries a turn's MCP bearer token.
	const mcpTokenEnv = "CALANGO_MCP_TOKEN"
	args := []string{
		"exec",
		"--ephemeral",
		"--ignore-user-config",
		"--ignore-rules",
		"--skip-git-repo-check",
		"--sandbox", "workspace-write",
		"-C", c.workDir,
		"-o", outPath,
		"-m", req.Model,
	}
	if req.Tools != nil {
		// The token travels in the environment; argv only names the variable.
		args = append(args,
			"-c", fmt.Sprintf("mcp_servers.calango.url=%q", req.Tools.URL),
			"-c", fmt.Sprintf("mcp_servers.calango.bearer_token_env_var=%q", mcpTokenEnv),
		)
	}
	args = append(args, "-")
	cmd := c.commandWithHome(ctx, home, args...)
	if req.Tools != nil {
		cmd.Env = append(cmd.Env, mcpTokenEnv+"="+req.Tools.Token)
	}
	cmd.Stdin = strings.NewReader(req.Prompt)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	// After cancellation, Wait closes the pipes after this delay, so a child
	// that outlives the CLI cannot keep Complete blocked.
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return gateway.Usage{}, fmt.Errorf("harness: codex run stopped: %w", ctxErr)
		}
		// The CLI's output is not included: it can echo the prompt.
		return gateway.Usage{}, fmt.Errorf("harness: codex exec failed: %w", err)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		return gateway.Usage{}, fmt.Errorf("harness: read codex message: %w", err)
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return gateway.Usage{}, errors.New("harness: codex finished without a message")
	}
	if stream != nil {
		stream(gateway.Chunk{Text: text})
	}
	return gateway.Usage{
		TokensIn:  estimateTokens(req.Prompt),
		TokensOut: estimateTokens(text),
		Estimated: true,
	}, nil
}

// estimateTokens uses the gateway's rough four bytes per token.
func estimateTokens(s string) int64 {
	return (int64(len(s)) + 3) / 4
}
