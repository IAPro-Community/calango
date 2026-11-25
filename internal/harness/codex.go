// Package harness runs model turns through agent CLIs. A CLI is a model
// provider behind the gateway: Calango builds the prompt, the gateway checks
// and records the budget, and the CLI runs the turn in an isolated process.
package harness

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"calango/internal/core"
	"calango/internal/gateway"

	"github.com/IAPro-Community/IAPro-SDK/providers"
	sdkcodex "github.com/IAPro-Community/IAPro-SDK/providers/codex"
	"github.com/IAPro-Community/IAPro-SDK/providers/process"
	sdkruntime "github.com/IAPro-Community/IAPro-SDK/runtime"
)

// ErrSignedOut means the CLI has no valid sign-in on this machine. The fix is
// to sign in again with the CLI, outside Calango.
var ErrSignedOut = errors.New("harness: CLI is signed out; sign in again with the CLI on this machine")

// ErrNotInstalled means the CLI binary was not found on PATH.
var ErrNotInstalled = errors.New("harness: CLI is not installed or not on PATH")

// ErrQuotaIdentityChanged is emitted only before a model dispatch.
var ErrQuotaIdentityChanged = errors.New("harness: quota identity changed before dispatch")

// ErrQuotaNotDispatched proves the model invocation was never started.
var ErrQuotaNotDispatched = process.ErrNotDispatched

// CodexName is the gateway provider name of the Codex CLI harness.
const CodexName = core.ProviderKindCodex

// Status says whether a CLI can run a turn right now.
type Status struct {
	Installed bool
	SignedIn  bool
}

// Codex runs turns through the OpenAI Codex CLI (`codex exec`). It implements
// gateway.Provider. Runs are isolated from the person's Codex configuration
// and session history; MCP calls remain subject to Calango's turn gateway.
type Codex struct {
	bin     string
	workDir string
	// home is the CODEX_HOME of this account. Empty means the person's default
	// Codex home, which is ADR 0013's shared default.
	home            string
	coordinateQuota bool
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
	overrides := make(map[string]string)
	if home != "" {
		overrides["CODEX_HOME"] = home
	}
	// Allow only platform variables; never inherit a previous turn's token.
	cmd.Env = sdkruntime.EnvSet(safeCLIEnv(os.Environ()), overrides, "CALANGO_MCP_TOKEN")
	return cmd
}

// CoordinateQuota serializes dedicated-home runs with quota maintenance.
func (c *Codex) CoordinateQuota() { c.coordinateQuota = true }

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
	signedIn, err := sdkcodex.CheckLoginStatus(ctx, func(runCtx context.Context) (*exec.Cmd, error) {
		return c.commandWithHome(runCtx, home, "login", "status"), nil
	})
	if err != nil {
		return Status{Installed: true}, fmt.Errorf("harness: codex login status: %w", err)
	}
	return Status{Installed: true, SignedIn: signedIn}, nil
}

// Complete implements gateway.Provider. The SDK decodes JSONL assistant messages
// as they complete and uses measured terminal usage when available. Calango
// retains the launch policy, MCP endpoint, account home and budget authority.
func (c *Codex) Complete(ctx context.Context, req gateway.Request, stream func(gateway.Chunk)) (gateway.Usage, error) {
	home := c.home
	if req.CodexHome != "" {
		home = req.CodexHome
	}
	if c.coordinateQuota && home != "" {
		release, err := lockQuotaHome(ctx, home)
		if err != nil {
			return gateway.Usage{}, errors.Join(ErrQuotaNotDispatched, err)
		}
		defer release()
	}
	status, err := c.available(ctx, home)
	if err != nil {
		return gateway.Usage{}, errors.Join(ErrQuotaNotDispatched, err)
	}
	if !status.Installed {
		return gateway.Usage{}, fmt.Errorf("%w: %s", ErrNotInstalled, c.bin)
	}
	if !status.SignedIn {
		return gateway.Usage{}, ErrSignedOut
	}

	if req.AdmissionScope != nil {
		adapter := QuotaAdapter{Codex: c}
		current, err := adapter.Identity(req.AdmissionOwner, req.AdmissionScope.ProfileID)
		if err != nil || current.Scope != *req.AdmissionScope || req.CodexHome != "" || req.CheckAdmission == nil {
			return gateway.Usage{}, ErrQuotaIdentityChanged
		}
		if err := req.CheckAdmission(ctx); err != nil {
			return gateway.Usage{}, ErrQuotaIdentityChanged
		}
	} else if req.CapacityRoute != nil {
		return gateway.Usage{}, ErrQuotaIdentityChanged
	}

	// mcpTokenEnv names the variable that carries a turn's MCP bearer token.
	const mcpTokenEnv = "CALANGO_MCP_TOKEN"
	args := []string{
		"exec",
		"--json",
		"--ephemeral",
		"--ignore-user-config",
		"--ignore-rules",
		"--skip-git-repo-check",
		"--sandbox", "workspace-write",
		"-C", c.workDir,
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
	var textBytes int64
	var messages int
	result, err := sdkcodex.RunExec(ctx, func(runCtx context.Context) (*exec.Cmd, error) {
		cmd := c.commandWithHome(runCtx, home, args...)
		if req.Tools != nil {
			cmd.Env = sdkruntime.EnvSet(cmd.Env, map[string]string{mcpTokenEnv: req.Tools.Token})
		}
		return cmd, nil
	}, req.Prompt, func(event providers.Event) error {
		if event.Kind != providers.EventText {
			return nil
		}
		text := strings.TrimSpace(event.Text)
		if text == "" {
			return nil
		}
		if messages > 0 {
			text = "\n\n" + text
		}
		messages++
		textBytes += int64(len(text))
		if stream != nil {
			stream(gateway.Chunk{Text: text})
		}
		return nil
	})
	// A failed turn with complete measured usage must still be recorded. If
	// usage is missing on failure, return zero so gateway charges its reservation.
	if result.InputTokens != nil && result.OutputTokens != nil {
		usage := gateway.Usage{TokensIn: *result.InputTokens, TokensOut: *result.OutputTokens}
		if err != nil {
			return usage, fmt.Errorf("harness: codex exec failed: %w", err)
		}
		if messages == 0 {
			return usage, errors.New("harness: codex finished without a message")
		}
		return usage, nil
	}
	if err != nil {
		return gateway.Usage{}, fmt.Errorf("harness: codex exec failed: %w", err)
	}
	if messages == 0 {
		return gateway.Usage{}, errors.New("harness: codex finished without a message")
	}
	return gateway.Usage{
		TokensIn:  estimateTokens(req.Prompt),
		TokensOut: (textBytes + 3) / 4,
		Estimated: true,
	}, nil
}

// estimateTokens uses the gateway's rough four bytes per token.
func estimateTokens(s string) int64 {
	return (int64(len(s)) + 3) / 4
}
