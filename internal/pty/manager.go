package pty

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

// Manager defaults.
const (
	DefaultMaxSessions = 8
	DefaultIdleTimeout = 30 * time.Minute
	DefaultScrollback  = 64 << 10
	// outputQueue is how many output chunks wait for a slow client before the reader
	// stops reading, which in turn stalls the program instead of dropping output.
	outputQueue = 64
	// drainGrace is how long a session keeps reading after its program exits, so the
	// last lines reach the client before the terminal is closed.
	drainGrace = 500 * time.Millisecond
	readChunk  = 32 << 10
)

// Close reasons reported to the audit hook.
const (
	ReasonExited     = "exited"
	ReasonTerminated = "terminated"
	ReasonIdle       = "idle"
	ReasonShutdown   = "shutdown"
)

// StartRequest is what a client asks for when it opens a terminal.
type StartRequest struct {
	OwnerID  string
	DeviceID string
	// Dir is the requested working directory, checked by the DirResolver.
	Dir        string
	Cols, Rows uint16
}

// StartPolicy decides whether a session may start. It is the hook for the terminal
// command policy (F4-SHELL, issue #89): sessions opened by a person pass today; an
// agent-driven terminal will be checked against the allowlist and approvals here.
// Returning an error that wraps ErrDenied refuses the session.
type StartPolicy interface {
	AllowStart(ctx context.Context, req StartRequest, shell string) error
}

// Info describes a session. It never carries terminal content.
type Info struct {
	ID        string
	OwnerID   string
	DeviceID  string
	Dir       string
	Shell     string
	Pid       int
	Cols      uint16
	Rows      uint16
	CreatedAt time.Time
	Attached  bool
}

// Event is reported to Config.OnEvent when a session opens or closes.
type Event struct {
	Action   string // "terminal.opened" or "terminal.closed"
	Info     Info
	Reason   string // close reason; empty on open
	ExitCode int    // program exit code on close; -1 when killed
}

// Event actions.
const (
	ActionOpened = "terminal.opened"
	ActionClosed = "terminal.closed"
)

// Config wires a Manager.
type Config struct {
	// Dirs resolves and restricts the working directory. Required.
	Dirs DirResolver
	// Shell returns the program and arguments to run. Defaults to DefaultShell.
	Shell func() (string, []string)
	// Env is the full child environment. Defaults to FilterEnv(os.Environ()).
	Env []string
	// MaxSessions bounds live sessions on the node. Defaults to DefaultMaxSessions.
	MaxSessions int
	// IdleTimeout closes a session with no client attached and no input for this
	// long. Defaults to DefaultIdleTimeout.
	IdleTimeout time.Duration
	// Scrollback is how many bytes of recent output are replayed on attach.
	Scrollback int
	// Policy may refuse a session before it starts. Nil allows every session.
	Policy StartPolicy
	// OnEvent receives open and close events, for the audit trail. It must not block.
	OnEvent func(Event)
	// Now is the clock. Defaults to time.Now.
	Now func() time.Time
	// Start launches the process. Defaults to Start; tests may replace it.
	Start func(Options) (Process, error)
}

// Manager owns the node's terminal sessions. Sessions live in memory only and die
// with the daemon.
type Manager struct {
	cfg Config

	mu       sync.Mutex
	sessions map[string]*session
	closed   bool

	stop     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// NewManager validates cfg and starts the idle sweeper.
func NewManager(cfg Config) (*Manager, error) {
	if cfg.Dirs == nil {
		return nil, errors.New("pty: directory resolver is required")
	}
	if cfg.Shell == nil {
		cfg.Shell = DefaultShell
	}
	if cfg.Env == nil {
		cfg.Env = FilterEnv(os.Environ())
	}
	if cfg.MaxSessions <= 0 {
		cfg.MaxSessions = DefaultMaxSessions
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = DefaultIdleTimeout
	}
	if cfg.Scrollback <= 0 {
		cfg.Scrollback = DefaultScrollback
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Start == nil {
		cfg.Start = Start
	}
	m := &Manager{cfg: cfg, sessions: map[string]*session{}, stop: make(chan struct{})}
	m.wg.Add(1)
	go m.sweepLoop()
	return m, nil
}

func (m *Manager) sweepLoop() {
	defer m.wg.Done()
	period := m.cfg.IdleTimeout / 4
	if period > time.Minute {
		period = time.Minute
	}
	if period < time.Second {
		period = time.Second
	}
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			m.Sweep()
		}
	}
}

// Sweep closes every session that has been idle for IdleTimeout and returns how many
// it closed. A session is idle when no client is attached and nobody typed or resized.
func (m *Manager) Sweep() int {
	now := m.cfg.Now()
	m.mu.Lock()
	var idle []*session
	for _, s := range m.sessions {
		if s != nil && s.idleSince(now) >= m.cfg.IdleTimeout {
			idle = append(idle, s)
		}
	}
	m.mu.Unlock()
	for _, s := range idle {
		s.terminate(ReasonIdle)
	}
	for _, s := range idle {
		<-s.done
	}
	return len(idle)
}

// Create starts a shell for req and returns its session.
func (m *Manager) Create(ctx context.Context, req StartRequest) (Info, error) {
	if req.OwnerID == "" || req.DeviceID == "" {
		return Info{}, fmt.Errorf("%w: owner and device are required", ErrInvalidInput)
	}
	if req.Cols == 0 && req.Rows == 0 {
		req.Cols, req.Rows = DefaultCols, DefaultRows
	}
	if err := validateSize(req.Cols, req.Rows); err != nil {
		return Info{}, err
	}
	dir, err := m.cfg.Dirs.ResolveDir(ctx, req.OwnerID, req.Dir)
	if err != nil {
		return Info{}, err
	}
	shell, args := m.cfg.Shell()
	if m.cfg.Policy != nil {
		if err := m.cfg.Policy.AllowStart(ctx, req, shell); err != nil {
			return Info{}, err
		}
	}
	id, err := newSessionID()
	if err != nil {
		return Info{}, err
	}

	// Reserve the slot before starting the process, so concurrent creates cannot
	// overshoot the limit.
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return Info{}, ErrClosed
	}
	if len(m.sessions) >= m.cfg.MaxSessions {
		m.mu.Unlock()
		return Info{}, ErrLimit
	}
	m.sessions[id] = nil
	m.mu.Unlock()

	proc, err := m.cfg.Start(Options{Path: shell, Args: args, Dir: dir, Env: m.cfg.Env, Cols: req.Cols, Rows: req.Rows})
	if err != nil {
		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()
		return Info{}, err
	}
	now := m.cfg.Now()
	s := &session{
		m: m,
		info: Info{
			ID: id, OwnerID: req.OwnerID, DeviceID: req.DeviceID, Dir: dir, Shell: shell,
			Pid: proc.Pid(), CreatedAt: now,
		},
		proc:       proc,
		cols:       req.Cols,
		rows:       req.Rows,
		lastActive: now,
		done:       make(chan struct{}),
		killed:     make(chan struct{}),
		pumped:     make(chan struct{}),
		exitCode:   -1,
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		_ = proc.Close()
		return Info{}, ErrClosed
	}
	m.sessions[id] = s
	m.mu.Unlock()

	m.wg.Add(2)
	go s.pump()
	go s.watch()
	m.emit(Event{Action: ActionOpened, Info: s.snapshot()})
	return s.snapshot(), nil
}

// List returns the caller's sessions, oldest first.
func (m *Manager) List(ownerID, deviceID string) []Info {
	m.mu.Lock()
	var out []Info
	for _, s := range m.sessions {
		if s != nil && s.info.OwnerID == ownerID && s.info.DeviceID == deviceID {
			out = append(out, s.snapshot())
		}
	}
	m.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// lookup returns a session owned by the caller. A session of another owner or device
// is reported as not found, so ids cannot be probed.
func (m *Manager) lookup(ownerID, deviceID, id string) (*session, error) {
	m.mu.Lock()
	s := m.sessions[id]
	m.mu.Unlock()
	if s == nil || s.info.OwnerID != ownerID || s.info.DeviceID != deviceID {
		return nil, ErrNotFound
	}
	return s, nil
}

// Attach connects one client to a session. Recent output is replayed first. Only one
// client may be attached at a time.
func (m *Manager) Attach(ownerID, deviceID, id string) (*Attachment, error) {
	s, err := m.lookup(ownerID, deviceID, id)
	if err != nil {
		return nil, err
	}
	return s.attach()
}

// Terminate kills a session and its whole process tree.
func (m *Manager) Terminate(ownerID, deviceID, id string) error {
	s, err := m.lookup(ownerID, deviceID, id)
	if err != nil {
		return err
	}
	s.terminate(ReasonTerminated)
	<-s.done
	return nil
}

// Close kills every session and waits for their goroutines. New sessions are refused.
func (m *Manager) Close() error {
	m.mu.Lock()
	m.closed = true
	var all []*session
	for _, s := range m.sessions {
		if s != nil {
			all = append(all, s)
		}
	}
	m.mu.Unlock()
	m.stopOnce.Do(func() { close(m.stop) })
	for _, s := range all {
		s.terminate(ReasonShutdown)
	}
	m.wg.Wait()
	return nil
}

func (m *Manager) remove(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

func (m *Manager) emit(ev Event) {
	if m.cfg.OnEvent != nil {
		m.cfg.OnEvent(ev)
	}
}

// newSessionID returns 256 random bits in URL-safe base64. The id is not a credential
// on its own (every route also needs the bearer token), but it must not be guessable.
func newSessionID() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("pty: session id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// ValidID reports whether id has the shape newSessionID produces.
func ValidID(id string) bool {
	if len(id) != 43 {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(id)
	return err == nil
}

type session struct {
	m    *Manager
	info Info // immutable after Create
	proc Process

	mu         sync.Mutex
	cols, rows uint16
	scrollback []byte
	client     *Attachment
	lastActive time.Time
	exitCode   int
	reason     string

	killOnce sync.Once
	killed   chan struct{} // closed when the session is being torn down
	pumped   chan struct{} // closed when the output reader stops
	done     chan struct{} // closed when teardown is complete
}

func (s *session) snapshot() Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	info := s.info
	info.Cols, info.Rows = s.cols, s.rows
	info.Attached = s.client != nil
	return info
}

func (s *session) idleSince(now time.Time) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		return 0
	}
	return now.Sub(s.lastActive)
}

func (s *session) touch() {
	now := s.m.cfg.Now()
	s.mu.Lock()
	s.lastActive = now
	s.mu.Unlock()
}

// pump copies program output to the scrollback and to the attached client. A slow
// client blocks the pump, which applies back-pressure to the program.
func (s *session) pump() {
	defer s.m.wg.Done()
	defer close(s.pumped)
	buf := make([]byte, readChunk)
	for {
		n, err := s.proc.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			s.mu.Lock()
			s.scrollback = appendBounded(s.scrollback, chunk, s.m.cfg.Scrollback)
			client := s.client
			s.mu.Unlock()
			if client != nil {
				select {
				case client.out <- chunk:
				case <-client.detached:
				case <-s.killed:
				}
			}
		}
		if err != nil {
			return
		}
	}
}

// watch tears the session down when the program exits.
func (s *session) watch() {
	defer s.m.wg.Done()
	select {
	case <-s.proc.Done():
		// Let the last output through before the terminal closes.
		select {
		case <-s.pumped:
		case <-time.After(drainGrace):
		case <-s.killed:
		}
		s.terminate(ReasonExited)
	case <-s.killed:
	}
}

// terminate kills the process tree once, waits for the reader, removes the session
// and reports it. Safe to call from any goroutine and more than once.
func (s *session) terminate(reason string) {
	s.killOnce.Do(func() {
		// The watcher is still counted in wg here, so this Add never races Wait at zero.
		s.m.wg.Add(1)
		s.mu.Lock()
		s.reason = reason
		s.mu.Unlock()
		close(s.killed)
		_ = s.proc.Close()
		go func() {
			defer s.m.wg.Done()
			<-s.pumped
			code := -1
			if reason == ReasonExited {
				code = s.proc.ExitCode()
			}
			s.mu.Lock()
			s.exitCode = code
			s.mu.Unlock()
			s.m.remove(s.info.ID)
			info := s.snapshot()
			close(s.done)
			s.m.emit(Event{Action: ActionClosed, Info: info, Reason: reason, ExitCode: code})
		}()
	})
}

func (s *session) attach() (*Attachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.killed:
		return nil, ErrNotFound
	default:
	}
	if s.client != nil {
		return nil, ErrBusy
	}
	a := &Attachment{s: s, out: make(chan []byte, outputQueue), detached: make(chan struct{})}
	if len(s.scrollback) > 0 {
		a.out <- append([]byte(nil), s.scrollback...)
	}
	s.client = a
	s.lastActive = s.m.cfg.Now()
	return a, nil
}

// appendBounded appends chunk and keeps only the last limit bytes.
func appendBounded(buf, chunk []byte, limit int) []byte {
	buf = append(buf, chunk...)
	if over := len(buf) - limit; over > 0 {
		buf = append(buf[:0], buf[over:]...)
	}
	return buf
}

// Attachment is one client connected to a session.
type Attachment struct {
	s        *session
	out      chan []byte
	detached chan struct{}
	once     sync.Once
}

// Output delivers program output: the scrollback first, then live output.
func (a *Attachment) Output() <-chan []byte { return a.out }

// Done is closed when the session has ended. Output may still hold the last chunks.
func (a *Attachment) Done() <-chan struct{} { return a.s.done }

// ExitCode is the program's exit code once Done is closed; -1 when it was killed.
func (a *Attachment) ExitCode() int {
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	return a.s.exitCode
}

// Reason is why the session ended, once Done is closed.
func (a *Attachment) Reason() string {
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	return a.s.reason
}

// Write types input into the terminal.
func (a *Attachment) Write(p []byte) (int, error) {
	select {
	case <-a.detached:
		return 0, ErrNotFound
	default:
	}
	a.s.touch()
	return a.s.proc.Write(p)
}

// Resize changes the terminal size.
func (a *Attachment) Resize(cols, rows uint16) error {
	if err := validateSize(cols, rows); err != nil {
		return err
	}
	if err := a.s.proc.Resize(cols, rows); err != nil {
		return err
	}
	a.s.mu.Lock()
	a.s.cols, a.s.rows = cols, rows
	a.s.lastActive = a.s.m.cfg.Now()
	a.s.mu.Unlock()
	return nil
}

// Detach disconnects the client and leaves the session running until it is idle for
// IdleTimeout. Idempotent.
func (a *Attachment) Detach() {
	a.once.Do(func() {
		close(a.detached)
		a.s.mu.Lock()
		if a.s.client == a {
			a.s.client = nil
		}
		a.s.lastActive = a.s.m.cfg.Now()
		a.s.mu.Unlock()
	})
}
