package harness

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"calango/internal/gateway"
)

// fakeCodex writes a stand-in for the codex binary. Its behaviour is driven by
// marker files in its isolated test directory, not inherited environment variables.
// It records its argv and stdin next to itself.
func fakeCodex(t *testing.T) (bin, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake codex is a shell script")
	}
	dir = t.TempDir()
	script := `#!/bin/sh
dir="$(dirname "$0")"
case "$1" in
login)
  printf '%s' "$CODEX_HOME" > "$dir/login_home.txt"
  if [ -f "$dir/FAKE_SIGNED_OUT" ]; then echo "Not logged in"; exit 1; fi
  echo "Logged in using ChatGPT"; exit 0;;
exec)
  printf '%s\n' "$@" > "$dir/args.txt"
  cat > "$dir/stdin.txt"
  printf '%s' "$CODEX_HOME" > "$dir/home.txt"
  printf '%s' "$CALANGO_MCP_TOKEN" > "$dir/mcp_token.txt"
  out=""
  while [ $# -gt 0 ]; do
    if [ "$1" = "-o" ]; then out="$2"; fi
    shift
  done
  if [ -f "$dir/FAKE_HANG" ]; then sleep 30; fi
  if [ -f "$dir/FAKE_EXEC_FAIL" ]; then echo "boom with secret-prompt" >&2; exit 3; fi
  echo "hello from fake codex" > "$out"; exit 0;;
esac
exit 2
`
	bin = filepath.Join(dir, "codex")
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, dir
}

func newFakeCodex(t *testing.T) (*Codex, string) {
	t.Helper()
	bin, dir := fakeCodex(t)
	c, err := NewCodex(bin, t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	return c, dir
}

func request(prompt string) gateway.Request {
	return gateway.Request{Provider: CodexName, Model: "gpt-test", Prompt: prompt, MaxOutputTokens: 100}
}

func TestCodexStreamsFinalMessageAndEstimatesUsage(t *testing.T) {
	c, _ := newFakeCodex(t)

	var got []string
	usage, err := c.Complete(context.Background(), request("say hi"), func(ch gateway.Chunk) {
		got = append(got, ch.Text)
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if len(got) != 1 || got[0] != "hello from fake codex" {
		t.Fatalf("streamed chunks = %q, want one final message", got)
	}
	if !usage.Estimated || usage.TokensIn <= 0 || usage.TokensOut <= 0 {
		t.Fatalf("usage = %+v, want estimated positive tokens", usage)
	}
}

func TestCodexUsesPerRunHomeOverrideForLoginAndExecution(t *testing.T) {
	c, dir := newFakeCodex(t)
	home := filepath.Join(t.TempDir(), "maestro-codex")
	req := request("say hi")
	req.CodexHome = home

	if _, err := c.Complete(context.Background(), req, nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	for _, name := range []string{"login_home.txt", "home.txt"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != home {
			t.Errorf("%s = %q, want %q", name, got, home)
		}
	}
}

func TestCodexRunsIsolatedAndSendsPromptOnStdin(t *testing.T) {
	c, dir := newFakeCodex(t)
	prompt := "secret user prompt"

	if _, err := c.Complete(context.Background(), request(prompt), nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	argv, err := os.ReadFile(filepath.Join(dir, "args.txt"))
	if err != nil {
		t.Fatal(err)
	}
	args := string(argv)
	for _, flag := range []string{"--ephemeral", "--ignore-user-config", "--ignore-rules", "--sandbox", "workspace-write"} {
		if !strings.Contains(args, flag) {
			t.Errorf("argv lacks %q: %s", flag, args)
		}
	}
	if strings.Contains(args, "secret user prompt") {
		t.Fatalf("prompt leaked into argv: %s", args)
	}

	stdin, err := os.ReadFile(filepath.Join(dir, "stdin.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(stdin) != prompt {
		t.Fatalf("stdin = %q, want the prompt", stdin)
	}
}

// The MCP endpoint reaches Codex as -c overrides. The bearer token reaches the
// child only through its environment, never through argv.
func TestCodexGetsMCPEndpointWithTokenInEnvironmentOnly(t *testing.T) {
	c, dir := newFakeCodex(t)
	req := request("hi")
	req.Tools = &gateway.ToolEndpoint{URL: "http://127.0.0.1:7317/mcp", Token: "turn-token-123"}

	if _, err := c.Complete(context.Background(), req, nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	argv, err := os.ReadFile(filepath.Join(dir, "args.txt"))
	if err != nil {
		t.Fatal(err)
	}
	args := string(argv)
	for _, want := range []string{
		`mcp_servers.calango.url="http://127.0.0.1:7317/mcp"`,
		`mcp_servers.calango.bearer_token_env_var="CALANGO_MCP_TOKEN"`,
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("argv lacks %s: %s", want, args)
		}
	}
	if strings.Contains(args, "turn-token-123") {
		t.Fatalf("MCP token leaked into argv: %s", args)
	}
	seen, err := os.ReadFile(filepath.Join(dir, "mcp_token.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(seen) != "turn-token-123" {
		t.Fatalf("child saw token %q, want the turn token", seen)
	}
}

// execCodexHome runs one exec through a fake codex built with home and returns
// the CODEX_HOME the child process saw.
func execCodexHome(t *testing.T, home string) string {
	t.Helper()
	bin, dir := fakeCodex(t)
	c, err := NewCodex(bin, t.TempDir(), home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Complete(context.Background(), request("hi"), nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	seen, err := os.ReadFile(filepath.Join(dir, "home.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(seen)
}

func TestCodexAccountHomeOverridesInheritedCodexHome(t *testing.T) {
	t.Setenv("CODEX_HOME", "/inherited")
	home := filepath.Join(t.TempDir(), "work")
	if got := execCodexHome(t, home); got != home {
		t.Fatalf("CODEX_HOME = %q, want the account home %q", got, home)
	}
}

func TestCodexDefaultAccountKeepsInheritedCodexHome(t *testing.T) {
	t.Setenv("CODEX_HOME", "/inherited")
	if got := execCodexHome(t, ""); got != "/inherited" {
		t.Fatalf("CODEX_HOME = %q, want the inherited default", got)
	}
}

func TestCodexSignedOutFailsBeforeRunningExec(t *testing.T) {
	c, dir := newFakeCodex(t)
	if err := os.WriteFile(filepath.Join(dir, "FAKE_SIGNED_OUT"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := c.Complete(context.Background(), request("hi"), nil)
	if !errors.Is(err, ErrSignedOut) {
		t.Fatalf("err = %v, want ErrSignedOut", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "args.txt")); statErr == nil {
		t.Fatal("codex exec ran while signed out")
	}

	status, err := c.Available(context.Background())
	if err != nil || !status.Installed || status.SignedIn {
		t.Fatalf("Available = %+v, %v; want installed and signed out", status, err)
	}
}

func TestCodexNotInstalled(t *testing.T) {
	c, err := NewCodex(filepath.Join(t.TempDir(), "no-such-codex"), t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	status, err := c.Available(context.Background())
	if err != nil || status.Installed {
		t.Fatalf("Available = %+v, %v; want not installed", status, err)
	}
	if _, err := c.Complete(context.Background(), request("hi"), nil); !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("err = %v, want ErrNotInstalled", err)
	}
}

func TestCodexExecFailureStreamsNothingAndDoesNotEchoOutput(t *testing.T) {
	c, dir := newFakeCodex(t)
	if err := os.WriteFile(filepath.Join(dir, "FAKE_EXEC_FAIL"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	var chunks int
	_, err := c.Complete(context.Background(), request("hi"), func(gateway.Chunk) { chunks++ })
	if err == nil {
		t.Fatal("expected an error from a failing codex exec")
	}
	if chunks != 0 {
		t.Fatalf("streamed %d chunks from a failed run", chunks)
	}
	if strings.Contains(err.Error(), "secret-prompt") {
		t.Fatalf("error echoes CLI output: %v", err)
	}
}

func TestCodexCancellationStopsTheRun(t *testing.T) {
	c, dir := newFakeCodex(t)
	if err := os.WriteFile(filepath.Join(dir, "FAKE_HANG"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.Complete(ctx, request("hi"), nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("cancelled run took %s; the CLI was not stopped", elapsed)
	}
}

func TestNewCodexRequiresWorkDir(t *testing.T) {
	if _, err := NewCodex("", "", ""); err == nil {
		t.Fatal("expected an error without a working directory")
	}
}
