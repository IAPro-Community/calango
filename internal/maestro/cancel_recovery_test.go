package maestro

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

type cancelAfterActivation struct {
	fakeRunner
	cancel    context.CancelFunc
	root      string
	target    string
	recovered bool
}

func (r *cancelAfterActivation) Run(ctx context.Context, dir string, env []string, executable string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out, err := r.fakeRunner.Run(ctx, dir, env, executable, args...)
	if err == nil && len(args) >= 3 && args[0] == "verify" && args[2] == r.root {
		if r.cancel != nil && strings.Contains(executable, filepath.Join("versions", r.target, "node_modules")) {
			r.cancel()
			r.cancel = nil
		} else if r.target != "" {
			r.recovered = true
		}
	}
	return out, err
}

func TestCancelledRollbackRestoresSelectedHome(t *testing.T) {
	runner := &cancelAfterActivation{}
	m, err := New(Config{DataDir: t.TempDir(), Runner: runner, Registry: fakeRegistry{}})
	if err != nil {
		t.Fatal(err)
	}
	runner.root = m.root
	ctx := context.Background()
	for _, version := range []string{"1.1.1", "1.1.2"} {
		if err := m.Install(ctx, version); err != nil {
			t.Fatal(err)
		}
	}
	rollbackCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	runner.target = "1.1.1"
	runner.cancel = cancel
	if err := m.Rollback(rollbackCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Rollback = %v, want cancellation", err)
	}
	if !runner.recovered {
		t.Fatal("selected release was not restored using a live context")
	}
	if got := lastHomeInstall(runner.calls, m.root); got != m.cliPath("1.1.2") {
		t.Fatalf("last installed home = %q, want selected release", got)
	}
	if status, err := m.Status(ctx); err != nil || status.Version != "1.1.2" {
		t.Fatalf("status = %+v, %v", status, err)
	}
}
