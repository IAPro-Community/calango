package maestro

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls      [][]string
	failVerify string
	activeEnv  []string
}

func (r *fakeRunner) Run(_ context.Context, _ string, env []string, executable string, args ...string) ([]byte, error) {
	call := append([]string{executable}, args...)
	r.calls = append(r.calls, call)
	r.activeEnv = append([]string(nil), env...)
	if strings.Contains(executable, filepath.Join(".bin", "orquestrador-maestro")) && len(args) > 0 && args[0] == "verify" && strings.Contains(executable, filepath.Join("versions", r.failVerify, "node_modules")) {
		return []byte("verify failed"), errors.New("verify failed")
	}
	return []byte("ok"), nil
}

type fakeRegistry []string

func (r fakeRegistry) Versions(context.Context) ([]string, error) {
	return append([]string(nil), r...), nil
}

func newTestManager(t *testing.T, runner *fakeRunner, versions []string, active func(context.Context) (bool, error), audit Audit) *Manager {
	t.Helper()
	m, err := New(Config{DataDir: t.TempDir(), Runner: runner, Registry: fakeRegistry(versions), ActiveRuns: active, Audit: audit})
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func TestCheckUpdateOnlyReturnsNewestStableV1(t *testing.T) {
	m := newTestManager(t, &fakeRunner{}, []string{"1.1.1", "1.2.0", "2.0.0", "1.3.0-beta.1"}, nil, nil)
	release, err := m.CheckUpdate(context.Background())
	if err != nil || release == nil || release.Version != "1.2.0" {
		t.Fatalf("release=%+v err=%v", release, err)
	}
	if err := m.Install(context.Background(), "2.0.0"); err == nil {
		t.Fatal("major version install was allowed")
	}
}
func TestInstallVerifiesBeforeSwitchAndRollback(t *testing.T) {
	runner := &fakeRunner{}
	var audits []string
	m := newTestManager(t, runner, nil, nil, func(_ context.Context, from, to string, auto bool, result string) error {
		audits = append(audits, from+":"+to+":"+result)
		return nil
	})

	status, err := m.Status(context.Background())
	if err != nil || !status.AutoUpdate || status.Installed || status.RollbackAvailable {
		t.Fatalf("initial status = %+v, err = %v", status, err)
	}
	if err := m.Install(context.Background(), "1.1.1"); err != nil {
		t.Fatal(err)
	}
	status, err = m.Status(context.Background())
	if err != nil || !status.Installed || status.RollbackAvailable {
		t.Fatalf("first install status = %+v, err = %v", status, err)
	}
	if err := m.Rollback(context.Background()); !errors.Is(err, ErrNoRollback) {
		t.Fatalf("rollback without a previous release = %v, want ErrNoRollback", err)
	}

	runner.failVerify = "1.1.2"
	if err := m.Install(context.Background(), "1.1.2"); err == nil {
		t.Fatal("failed verification accepted")
	}
	status, err = m.Status(context.Background())
	if err != nil || status.Version != "1.1.1" || status.RollbackAvailable {
		t.Fatalf("after failed verify status = %+v, err = %v", status, err)
	}

	runner.failVerify = ""
	if err := m.Install(context.Background(), "1.1.2"); err != nil {
		t.Fatal(err)
	}
	status, err = m.Status(context.Background())
	if err != nil || status.Version != "1.1.2" || !status.RollbackAvailable {
		t.Fatalf("after update status = %+v, err = %v", status, err)
	}
	if err := m.Install(context.Background(), "1.1.2"); err != nil {
		t.Fatal(err)
	}
	if err := m.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	status, err = m.Status(context.Background())
	if err != nil || status.Version != "1.1.1" || !status.RollbackAvailable {
		t.Fatalf("after rollback status = %+v, err = %v", status, err)
	}
	if err := m.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	status, err = m.Status(context.Background())
	if err != nil || status.Version != "1.1.2" {
		t.Fatalf("after rollback toggle status = %+v, err = %v", status, err)
	}
	if len(audits) != 5 || !strings.Contains(audits[1], "verify_failed") {
		t.Fatalf("audit records = %v", audits)
	}
	for _, required := range []string{
		"ORQUESTRADOR_MAESTRO_SKIP_CLI_UPDATE=1",
		"ORQUESTRADOR_MAESTRO_TELEMETRY=0",
		"XDG_CONFIG_HOME=",
		"CODEX_HOME=",
	} {
		found := false
		for _, value := range runner.activeEnv {
			if strings.HasPrefix(value, required) {
				found = true
			}
		}
		if !found {
			t.Errorf("missing isolated environment %q", required)
		}
	}
}
func TestAutomaticUpdateSettingPersistsAndActiveRunBlocksChanges(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{}
	busy := true
	m, err := New(Config{DataDir: root, Runner: runner, Registry: fakeRegistry{"1.1.1"}, ActiveRuns: func(context.Context) (bool, error) { return busy, nil }})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetAutoUpdate(false); err != nil {
		t.Fatal(err)
	}
	m2, err := New(Config{DataDir: root, Runner: runner, Registry: fakeRegistry{"1.1.1"}})
	if err != nil {
		t.Fatal(err)
	}
	status, err := m2.Status(context.Background())
	if err != nil || status.AutoUpdate {
		t.Fatalf("persisted status=%+v err=%v", status, err)
	}
	if err := m.Install(context.Background(), "1.1.1"); err == nil {
		t.Fatal("install proceeded during active run")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("commands ran during active run: %v", runner.calls)
	}
}

func TestCodexHomeRequiresActiveVerifiedRelease(t *testing.T) {
	m := newTestManager(t, &fakeRunner{}, nil, nil, nil)
	if _, err := m.CodexHome(); err == nil {
		t.Fatal("Codex home was returned without an active verified release")
	}
	if err := m.Install(context.Background(), "1.1.1"); err != nil {
		t.Fatal(err)
	}
	home, err := m.CodexHome()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(m.root, ".codex")
	if home != want {
		t.Fatalf("Codex home = %q, want %q", home, want)
	}
}

func TestCodexHomeIsStableAcrossReleases(t *testing.T) {
	runner := &fakeRunner{}
	auditFailure := ""
	m := newTestManager(t, runner, nil, nil, func(_ context.Context, _, _ string, _ bool, result string) error {
		if result == auditFailure {
			return errors.New("audit store unavailable")
		}
		return nil
	})
	ctx := context.Background()
	if err := m.Install(ctx, "1.1.1"); err != nil {
		t.Fatal(err)
	}
	home, err := m.CodexHome()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Install(ctx, "1.1.2"); err != nil {
		t.Fatal(err)
	}
	if after, err := m.CodexHome(); err != nil || after != home {
		t.Fatalf("Codex home after update = %q, err = %v; want %q", after, err, home)
	}

	// A failed audit after the switch must re-activate the release that stays selected.
	auditFailure = "rollback"
	runner.calls = nil
	if err := m.Rollback(ctx); err == nil {
		t.Fatal("rollback succeeded after its audit write failed")
	}
	if got := lastHomeInstall(runner.calls, m.root); got != m.cliPath("1.1.2") {
		t.Fatalf("Codex home last installed by %q, want the selected release 1.1.2", got)
	}
}

func TestFailedReleaseVerificationNeverWritesActiveCodexHome(t *testing.T) {
	runner := &fakeRunner{}
	m := newTestManager(t, runner, nil, nil, nil)
	ctx := context.Background()
	if err := m.Install(ctx, "1.1.1"); err != nil {
		t.Fatal(err)
	}
	runner.calls = nil
	runner.failVerify = "1.1.2"
	if err := m.Install(ctx, "1.1.2"); err == nil {
		t.Fatal("failed verification accepted")
	}
	if got := lastHomeInstall(runner.calls, m.root); got != "" {
		t.Fatalf("active Codex home was written by %q during a failed staging verification", got)
	}
}

// lastHomeInstall returns the Maestro executable of the last install that targeted home.
func lastHomeInstall(calls [][]string, home string) string {
	for i := len(calls) - 1; i >= 0; i-- {
		call := calls[i]
		if len(call) > 3 && call[1] == "install" && call[3] == home {
			return call[0]
		}
	}
	return ""
}

func TestCleanupRetainsActiveAndPreviousPlusNewestVerifiedV1(t *testing.T) {
	m := newTestManager(t, &fakeRunner{}, nil, nil, nil)
	versions := []string{"1.1.1", "1.2.0", "1.9.0", "1.10.0", "1.11.0"}
	for _, version := range versions {
		dir := filepath.Join(m.root, "versions", version)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".verified"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	m.cleanup("1.1.1", "1.2.0")
	kept := map[string]bool{"1.1.1": true, "1.2.0": true, "1.11.0": true}
	for _, version := range versions {
		_, err := os.Stat(filepath.Join(m.root, "versions", version))
		if kept[version] && err != nil {
			t.Errorf("expected verified version %s to be retained: %v", version, err)
		}
		if !kept[version] && !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected old verified version %s to be removed, stat error = %v", version, err)
		}
	}
}

func TestAuditFailuresRestoreVersionPointers(t *testing.T) {
	auditFailure := ""
	m := newTestManager(t, &fakeRunner{}, nil, nil, func(_ context.Context, _, _ string, _ bool, result string) error {
		if result == auditFailure {
			return errors.New("audit store unavailable")
		}
		return nil
	})

	if err := m.Install(context.Background(), "1.1.1"); err != nil {
		t.Fatal(err)
	}
	auditFailure = "success"
	if err := m.Install(context.Background(), "1.1.2"); err == nil {
		t.Fatal("install succeeded after its audit write failed")
	}
	status, err := m.Status(context.Background())
	if err != nil || status.Version != "1.1.1" || status.RollbackAvailable {
		t.Fatalf("after failed install audit status = %+v, err = %v", status, err)
	}

	auditFailure = ""
	if err := m.Install(context.Background(), "1.1.2"); err != nil {
		t.Fatal(err)
	}
	auditFailure = "rollback"
	if err := m.Rollback(context.Background()); err == nil {
		t.Fatal("rollback succeeded after its audit write failed")
	}
	status, err = m.Status(context.Background())
	if err != nil || status.Version != "1.1.2" || !status.RollbackAvailable {
		t.Fatalf("after failed rollback audit status = %+v, err = %v", status, err)
	}

	auditFailure = ""
	if err := m.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
}
