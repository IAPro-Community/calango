//go:build unix

package pty

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// newTestManager runs /bin/sh in a temporary workspace. /bin/sh exists on every Unix
// CI runner; the person's $SHELL and profile are not involved.
func newTestManager(t *testing.T, mutate func(*Config)) (*Manager, string) {
	t.Helper()
	root := t.TempDir()
	dirs, err := NewRootDir(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Dirs:  dirs,
		Shell: func() (string, []string) { return "/bin/sh", nil },
		Env:   append(FilterEnv(os.Environ()), "PS1=$ "),
	}
	if mutate != nil {
		mutate(&cfg)
	}
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m, dirs.Root()
}

func create(t *testing.T, m *Manager, dir string) Info {
	t.Helper()
	info, err := m.Create(context.Background(), StartRequest{OwnerID: "owner", DeviceID: "dev", Dir: dir, Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return info
}

func attach(t *testing.T, m *Manager, id string) *Attachment {
	t.Helper()
	a, err := m.Attach("owner", "dev", id)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	t.Cleanup(a.Detach)
	return a
}

// readUntil collects output until it matches pattern or the deadline passes.
func readUntil(t *testing.T, a *Attachment, pattern string) []string {
	t.Helper()
	re := regexp.MustCompile(pattern)
	var buf bytes.Buffer
	deadline := time.After(10 * time.Second)
	for {
		if m := re.FindStringSubmatch(buf.String()); m != nil {
			return m
		}
		select {
		case chunk := <-a.Output():
			buf.Write(chunk)
		case <-deadline:
			t.Fatalf("output never matched %q; got %q", pattern, buf.String())
		}
	}
}

func write(t *testing.T, a *Attachment, s string) {
	t.Helper()
	if _, err := a.Write([]byte(s)); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestSessionEchoesInputAndRunsInTheWorkspace(t *testing.T) {
	m, root := newTestManager(t, nil)
	if err := os.Mkdir(filepath.Join(root, "proj"), 0o700); err != nil {
		t.Fatal(err)
	}
	info := create(t, m, "proj")
	if info.Dir != filepath.Join(root, "proj") || !ValidID(info.ID) || info.Pid <= 0 {
		t.Fatalf("info = %+v", info)
	}
	a := attach(t, m, info.ID)

	write(t, a, "echo sum-$((40+2)); pwd\n")
	readUntil(t, a, `(?s)sum-42.*`+regexp.QuoteMeta(filepath.Join(root, "proj")))
}

func TestSessionEnvironmentHasNoSecrets(t *testing.T) {
	t.Setenv("CALANGO_VAULT_KEY", "vault-canary-91")
	t.Setenv("GITHUB_TOKEN", "token-canary-92")
	m, _ := newTestManager(t, func(c *Config) { c.Env = nil }) // default: FilterEnv(os.Environ())
	a := attach(t, m, create(t, m, "").ID)

	write(t, a, "env; echo END-$((1+1))\n")
	out := readUntil(t, a, `(?s)(.*)END-2`)[1]
	if strings.Contains(out, "vault-canary-91") || strings.Contains(out, "token-canary-92") {
		t.Fatalf("terminal environment leaks a secret:\n%s", out)
	}
	if !strings.Contains(out, "TERM=xterm-256color") {
		t.Fatalf("TERM not set:\n%s", out)
	}
}

func TestResizeReachesTheProgram(t *testing.T) {
	m, _ := newTestManager(t, nil)
	a := attach(t, m, create(t, m, "").ID)

	if err := a.Resize(101, 37); err != nil {
		t.Fatal(err)
	}
	write(t, a, "stty size\n")
	readUntil(t, a, `37 101`)
	if err := a.Resize(0, 10); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("resize 0 cols err = %v", err)
	}
}

func TestCreateRejectsDirectoriesOutsideTheWorkspace(t *testing.T) {
	m, root := newTestManager(t, nil)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"/", "..", outside, "link", "missing"} {
		_, err := m.Create(context.Background(), StartRequest{OwnerID: "owner", DeviceID: "dev", Dir: dir})
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("dir %q: err = %v, want ErrInvalidInput", dir, err)
		}
	}
	if got := len(m.List("owner", "dev")); got != 0 {
		t.Fatalf("sessions after refusals = %d", got)
	}
}

func TestCreateRejectsBadSizes(t *testing.T) {
	m, _ := newTestManager(t, nil)
	for _, sz := range [][2]uint16{{0, 10}, {10, 0}, {MaxCols + 1, 10}, {10, MaxRows + 1}} {
		_, err := m.Create(context.Background(), StartRequest{OwnerID: "owner", DeviceID: "dev", Cols: sz[0], Rows: sz[1]})
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("size %v: err = %v", sz, err)
		}
	}
}

func TestSessionLimitIsEnforced(t *testing.T) {
	m, _ := newTestManager(t, func(c *Config) { c.MaxSessions = 2 })
	first := create(t, m, "")
	create(t, m, "")
	if _, err := m.Create(context.Background(), StartRequest{OwnerID: "owner", DeviceID: "dev"}); !errors.Is(err, ErrLimit) {
		t.Fatalf("third create err = %v, want ErrLimit", err)
	}
	if err := m.Terminate("owner", "dev", first.ID); err != nil {
		t.Fatal(err)
	}
	create(t, m, "")
}

func TestSessionsAreScopedToOwnerAndDevice(t *testing.T) {
	m, _ := newTestManager(t, nil)
	info := create(t, m, "")
	if _, err := m.Attach("owner", "other-device", info.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("attach from other device err = %v", err)
	}
	if err := m.Terminate("other", "dev", info.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("terminate from other owner err = %v", err)
	}
	if got := m.List("owner", "other-device"); len(got) != 0 {
		t.Fatalf("other device lists %d sessions", len(got))
	}
	attach(t, m, info.ID)
	if _, err := m.Attach("owner", "dev", info.ID); !errors.Is(err, ErrBusy) {
		t.Fatalf("second attach err = %v, want ErrBusy", err)
	}
}

func TestScrollbackIsReplayedOnAttach(t *testing.T) {
	m, _ := newTestManager(t, nil)
	info := create(t, m, "")
	a := attach(t, m, info.ID)
	write(t, a, "echo replay-$((2*21))\n")
	readUntil(t, a, "replay-42")
	a.Detach()

	b := attach(t, m, info.ID)
	readUntil(t, b, "replay-42")
}

// alive reports whether pid still runs. A zombie counts as gone.
func alive(pid int) bool {
	if err := syscall.Kill(pid, 0); err != nil {
		return false
	}
	if data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil {
		if i := bytes.LastIndexByte(data, ')'); i >= 0 && bytes.HasPrefix(bytes.TrimSpace(data[i+1:]), []byte("Z")) {
			return false
		}
	}
	return true
}

func waitGone(t *testing.T, pids ...int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for _, pid := range pids {
		for alive(pid) {
			if time.Now().After(deadline) {
				t.Fatalf("process %d still running after cleanup", pid)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// startBackgroundJob starts a long sleep as a background job (its own process group
// under job control) and returns its pid.
func startBackgroundJob(t *testing.T, a *Attachment) int {
	t.Helper()
	write(t, a, "sleep 300 & echo BG=$!\n")
	pid, err := strconv.Atoi(readUntil(t, a, `BG=(\d+)`)[1])
	if err != nil || !alive(pid) {
		t.Fatalf("background job pid %d not running (%v)", pid, err)
	}
	return pid
}

func TestTerminateKillsTheWholeTree(t *testing.T) {
	var mu sync.Mutex
	var events []Event
	m, _ := newTestManager(t, func(c *Config) {
		c.OnEvent = func(ev Event) { mu.Lock(); events = append(events, ev); mu.Unlock() }
	})
	info := create(t, m, "")
	a := attach(t, m, info.ID)
	bg := startBackgroundJob(t, a)

	if err := m.Terminate("owner", "dev", info.ID); err != nil {
		t.Fatal(err)
	}
	waitGone(t, info.Pid, bg)
	select {
	case <-a.Done():
	default:
		t.Fatal("attachment not done after terminate")
	}
	if a.ExitCode() != -1 || a.Reason() != ReasonTerminated {
		t.Fatalf("exit = %d reason = %q", a.ExitCode(), a.Reason())
	}
	if _, err := m.Attach("owner", "dev", info.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("attach after terminate err = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 || events[0].Action != ActionOpened || events[1].Action != ActionClosed || events[1].Reason != ReasonTerminated {
		t.Fatalf("events = %+v", events)
	}
}

func TestShellExitEndsSessionAndKillsLeftoverJobs(t *testing.T) {
	m, _ := newTestManager(t, nil)
	info := create(t, m, "")
	a := attach(t, m, info.ID)
	bg := startBackgroundJob(t, a)

	write(t, a, "exit 3\n")
	select {
	case <-a.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("session did not end after exit")
	}
	if a.ExitCode() != 3 || a.Reason() != ReasonExited {
		t.Fatalf("exit = %d reason = %q", a.ExitCode(), a.Reason())
	}
	waitGone(t, bg)
	if got := len(m.List("owner", "dev")); got != 0 {
		t.Fatalf("sessions after exit = %d", got)
	}
}

func TestCloseKillsEverySessionAndRefusesNewOnes(t *testing.T) {
	m, _ := newTestManager(t, nil)
	one := create(t, m, "")
	two := create(t, m, "")
	bg := startBackgroundJob(t, attach(t, m, two.ID))

	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	waitGone(t, one.Pid, two.Pid, bg)
	if _, err := m.Create(context.Background(), StartRequest{OwnerID: "owner", DeviceID: "dev"}); !errors.Is(err, ErrClosed) {
		t.Fatalf("create after close err = %v", err)
	}
}

func TestSweepClosesOnlyIdleDetachedSessions(t *testing.T) {
	var mu sync.Mutex
	now := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	advance := func(d time.Duration) { mu.Lock(); now = now.Add(d); mu.Unlock() }
	m, _ := newTestManager(t, func(c *Config) { c.Now = clock; c.IdleTimeout = 10 * time.Minute })

	idle := create(t, m, "")
	busy := create(t, m, "")
	attach(t, m, busy.ID)

	advance(9 * time.Minute)
	if n := m.Sweep(); n != 0 {
		t.Fatalf("swept %d before the timeout", n)
	}
	advance(2 * time.Minute)
	if n := m.Sweep(); n != 1 {
		t.Fatalf("swept %d, want 1", n)
	}
	waitGone(t, idle.Pid)
	left := m.List("owner", "dev")
	if len(left) != 1 || left[0].ID != busy.ID {
		t.Fatalf("left = %+v", left)
	}
}

type denyAll struct{}

func (denyAll) AllowStart(context.Context, StartRequest, string) error { return ErrDenied }

func TestStartPolicyCanRefuse(t *testing.T) {
	started := false
	m, _ := newTestManager(t, func(c *Config) {
		c.Policy = denyAll{}
		c.Start = func(Options) (Process, error) { started = true; return nil, errors.New("unreachable") }
	})
	if _, err := m.Create(context.Background(), StartRequest{OwnerID: "owner", DeviceID: "dev"}); !errors.Is(err, ErrDenied) {
		t.Fatalf("err = %v, want ErrDenied", err)
	}
	if started {
		t.Fatal("process started despite the policy")
	}
}
