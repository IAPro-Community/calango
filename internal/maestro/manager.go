// Package maestro manages the official Orquestrador Maestro CLI in Calango's data directory.
package maestro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
)

const (
	packageName = "@iapro/orquestrador-maestro-cli"
	pinnedLine  = "1.1.1"
)

var (
	ErrInvalidVersion = errors.New("maestro: only stable V1 versions may be installed")
	ErrActiveRun      = errors.New("maestro: cannot change versions while an agent run is active")
	ErrNoRollback     = errors.New("maestro: no previous verified version")
)

var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

// Runner executes npm and Maestro commands. Implementations must honor context cancellation.
type Runner interface {
	Run(context.Context, string, []string, string, ...string) ([]byte, error)
}

// Registry returns published package versions, as npm view does.
type Registry interface {
	Versions(context.Context) ([]string, error)
}

// Audit records a version transition and its result. Facts must not contain secrets.
type Audit func(context.Context, string, string, bool, string) error

// Config configures a manager rooted entirely under the Calango data directory.
type Config struct {
	DataDir    string
	Runner     Runner
	Registry   Registry
	Audit      Audit
	ActiveRuns func(context.Context) (bool, error)
}

// Manager controls isolated installs and the persisted automatic-update preference.
type Manager struct {
	root       string
	runner     Runner
	registry   Registry
	audit      Audit
	activeRuns func(context.Context) (bool, error)
	available  *Release
	mu         sync.Mutex
}

// Status describes the locally selected install and update preference.
type Status struct {
	Installed         bool     `json:"installed"`
	Version           string   `json:"version,omitempty"`
	AutoUpdate        bool     `json:"auto_update"`
	RollbackAvailable bool     `json:"rollback_available"`
	UpdateAvailable   *Release `json:"update_available,omitempty"`
}

// Release is the newest available version on the supported V1 line.
type Release struct {
	Version string `json:"version"`
}

// New creates a manager under <data-dir>/maestro.
func New(cfg Config) (*Manager, error) {
	if cfg.DataDir == "" {
		return nil, errors.New("maestro: data directory is required")
	}
	root, err := filepath.Abs(filepath.Join(cfg.DataDir, "maestro"))
	if err != nil {
		return nil, fmt.Errorf("maestro: resolve data directory: %w", err)
	}
	if cfg.Runner == nil {
		cfg.Runner = execRunner{}
	}
	if cfg.Registry == nil {
		cfg.Registry = npmRegistry{runner: cfg.Runner, root: root, env: []string{"HOME=" + root, "USERPROFILE=" + root, "XDG_CONFIG_HOME=" + filepath.Join(root, "config"), "NPM_CONFIG_USERCONFIG=" + filepath.Join(root, "npmrc"), "npm_config_cache=" + filepath.Join(root, "npm-cache"), "NPM_CONFIG_REGISTRY=https://registry.npmjs.org/", "NPM_CONFIG_UPDATE_NOTIFIER=false"}}
	}
	return &Manager{root: root, runner: cfg.Runner, registry: cfg.Registry, audit: cfg.Audit, activeRuns: cfg.ActiveRuns}, nil
}

// Status returns the active version, rollback availability and update preference.
func (m *Manager) Status(ctx context.Context) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	autoUpdate, err := m.autoUpdate()
	if err != nil {
		return Status{}, err
	}
	status := Status{AutoUpdate: autoUpdate}
	if m.available != nil {
		release := *m.available
		status.UpdateAvailable = &release
	}
	version, err := m.currentVersion()
	if errors.Is(err, os.ErrNotExist) {
		return status, nil
	}
	if err != nil {
		return Status{}, err
	}
	status.Installed, status.Version = true, version

	previous, err := m.previousVersion()
	if errors.Is(err, os.ErrNotExist) {
		return status, nil
	}
	if err != nil {
		return Status{}, err
	}
	if previous == version {
		return status, nil
	}
	_, err = os.Stat(filepath.Join(m.root, "versions", previous, ".verified"))
	if err == nil {
		status.RollbackAvailable = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return Status{}, err
	}
	return status, nil
}

// CodexHome returns the stable Codex home that every verified release is activated into.
func (m *Manager) CodexHome() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.currentVersion()
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("maestro: no active verified installation")
	}
	if err != nil {
		return "", err
	}
	return m.codexHome(), nil
}

// SetAutoUpdate persists the owner's automatic update preference.
func (m *Manager) SetAutoUpdate(enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeAutoUpdate(enabled)
}

// CheckUpdate checks only npm-published stable V1 versions.
func (m *Manager) CheckUpdate(ctx context.Context) (*Release, error) {
	versions, err := m.registry.Versions(ctx)
	if err != nil {
		return nil, fmt.Errorf("maestro: query npm registry: %w", err)
	}
	current, err := m.currentVersion()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	best := ""
	for _, candidate := range versions {
		major, _, _, stable := parseVersion(candidate)
		if !stable || major != 1 || compareVersions(candidate, pinnedLine) < 0 {
			continue
		}
		if best == "" || compareVersions(candidate, best) > 0 {
			best = candidate
		}
	}
	var release *Release
	if best != "" && best != current && (current == "" || compareVersions(best, current) >= 0) {
		release = &Release{Version: best}
	}
	m.mu.Lock()
	if release == nil {
		m.available = nil
	} else {
		value := *release
		m.available = &value
	}
	m.mu.Unlock()
	return release, nil
}

// Install downloads one explicit stable V1 release, verifies it, then atomically selects it.
func (m *Manager) Install(ctx context.Context, version string) error {
	return m.install(ctx, version, false)
}

func (m *Manager) install(ctx context.Context, version string, automatic bool) error {
	major, _, _, stable := parseVersion(version)
	if !stable || major != 1 || compareVersions(version, pinnedLine) < 0 {
		return fmt.Errorf("%w: %q", ErrInvalidVersion, version)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureNoActiveRuns(ctx); err != nil {
		return err
	}
	if err := os.MkdirAll(m.root, 0o700); err != nil {
		return fmt.Errorf("maestro: create home: %w", err)
	}
	versionsDir := filepath.Join(m.root, "versions")
	dest := filepath.Join(versionsDir, version)
	if err := os.MkdirAll(versionsDir, 0o700); err != nil {
		return fmt.Errorf("maestro: create versions directory: %w", err)
	}

	current, err := m.currentVersion()
	if errors.Is(err, os.ErrNotExist) {
		current = ""
	} else if err != nil {
		return err
	}
	if current == version {
		return nil
	}
	oldPrevious, err := m.previousVersion()
	if errors.Is(err, os.ErrNotExist) {
		oldPrevious = ""
	} else if err != nil {
		return err
	}

	if err := os.MkdirAll(dest, 0o700); err != nil {
		return fmt.Errorf("maestro: create version directory: %w", err)
	}
	npmArgs := []string{
		"install", "--prefix", dest, "--no-save", "--no-audit", "--no-fund",
		"--registry=https://registry.npmjs.org/", packageName + "@" + version,
	}
	if err := os.WriteFile(filepath.Join(m.root, "npmrc"), []byte("registry=https://registry.npmjs.org/\n"), 0o600); err != nil {
		return fmt.Errorf("maestro: configure npm registry: %w", err)
	}
	if _, err := m.runner.Run(ctx, m.root, m.env(m.root), "npm", npmArgs...); err != nil {
		return fmt.Errorf("maestro: install %s: %w", version, err)
	}

	cli := m.cliPath(version)
	staging := m.stagingHome(version)
	if _, err := m.runner.Run(ctx, m.root, m.env(staging), cli, "install", "--home-path", staging, "--only", "codex", "--non-interactive"); err != nil {
		return fmt.Errorf("maestro: configure %s: %w", version, err)
	}
	if _, err := m.runner.Run(ctx, m.root, m.env(staging), cli, "verify", "--home-path", staging); err != nil {
		auditErr := m.writeAudit(ctx, current, version, automatic, "verify_failed")
		return errors.Join(fmt.Errorf("maestro: verify %s: %w", version, err), auditErr)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dest, ".verified"), []byte("verified\n"), 0o600); err != nil {
		return fmt.Errorf("maestro: record verified version: %w", err)
	}

	if err := m.activate(ctx, version); err != nil {
		return errors.Join(fmt.Errorf("maestro: activate %s: %w", version, err), m.restoreHome(ctx, current))
	}
	if err := m.writePrevious(current); err != nil {
		return errors.Join(fmt.Errorf("maestro: record previous version: %w", err), m.restoreHome(ctx, current))
	}
	if err := m.writeCurrent(version); err != nil {
		return errors.Join(err, m.writePrevious(oldPrevious), m.restoreHome(ctx, current))
	}
	if err := m.writeAudit(ctx, current, version, automatic, "success"); err != nil {
		var restoreCurrentErr error
		if current != "" {
			restoreCurrentErr = m.writeCurrent(current)
		} else {
			restoreCurrentErr = os.Remove(filepath.Join(m.root, "current"))
		}
		return fmt.Errorf("maestro: audit version switch: %w", errors.Join(err, restoreCurrentErr, m.writePrevious(oldPrevious), m.restoreHome(ctx, current)))
	}

	m.available = nil
	m.cleanup(version, current)
	return nil
}

// Rollback verifies the previously active V1 release before selecting it.
func (m *Manager) Rollback(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureNoActiveRuns(ctx); err != nil {
		return err
	}
	current, err := m.currentVersion()
	if errors.Is(err, os.ErrNotExist) {
		return ErrNoRollback
	}
	if err != nil {
		return fmt.Errorf("maestro: no active version to roll back: %w", err)
	}
	previous, err := m.previousVersion()
	if errors.Is(err, os.ErrNotExist) {
		return ErrNoRollback
	}
	if err != nil {
		return err
	}
	if previous == current {
		return ErrNoRollback
	}
	if _, err := os.Stat(filepath.Join(m.root, "versions", previous, ".verified")); errors.Is(err, os.ErrNotExist) {
		return ErrNoRollback
	} else if err != nil {
		return fmt.Errorf("maestro: inspect previous version: %w", err)
	}
	if err := m.activate(ctx, previous); err != nil {
		return errors.Join(fmt.Errorf("maestro: previous version failed verification: %w", err), m.restoreHome(ctx, current))
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := m.writePrevious(current); err != nil {
		return errors.Join(fmt.Errorf("maestro: record previous version: %w", err), m.restoreHome(ctx, current))
	}
	if err := m.writeCurrent(previous); err != nil {
		return errors.Join(err, m.writePrevious(previous), m.restoreHome(ctx, current))
	}
	if err := m.writeAudit(ctx, current, previous, false, "rollback"); err != nil {
		return errors.Join(
			fmt.Errorf("maestro: audit rollback: %w", err),
			m.writeCurrent(current),
			m.writePrevious(previous),
			m.restoreHome(ctx, current),
		)
	}
	m.available = nil
	m.cleanup(previous, current)
	return nil
}

// Update checks the registry and installs the newest V1 release. It never invokes Maestro's self-updater.
func (m *Manager) Update(ctx context.Context) (*Release, error) {
	release, err := m.CheckUpdate(ctx)
	if err != nil || release == nil {
		return release, err
	}
	if err := m.install(ctx, release.Version, false); err != nil {
		return release, err
	}
	return release, nil
}

// RunAutoUpdate is called by the daemon's daily scheduler. Errors are returned for logging, not alerts.
func (m *Manager) RunAutoUpdate(ctx context.Context) (*Release, error) {
	status, err := m.Status(ctx)
	if err != nil {
		return nil, err
	}
	if !status.AutoUpdate {
		return nil, nil
	}
	release, err := m.CheckUpdate(ctx)
	if err != nil || release == nil {
		return release, err
	}
	if err := m.install(ctx, release.Version, true); err != nil {
		return release, err
	}
	return release, nil
}

func (m *Manager) ensureNoActiveRuns(ctx context.Context) error {
	if m.activeRuns == nil {
		return nil
	}
	active, err := m.activeRuns(ctx)
	if err != nil {
		return fmt.Errorf("maestro: check active runs: %w", err)
	}
	if active {
		return ErrActiveRun
	}
	return nil
}
func (m *Manager) writeAudit(ctx context.Context, from, to string, automatic bool, result string) error {
	if m.audit == nil {
		return nil
	}
	return m.audit(ctx, from, to, automatic, result)
}

// env isolates Maestro under home. USERPROFILE gets the same value as HOME because on
// Windows Node and PowerShell resolve ~ from USERPROFILE; with only HOME isolated,
// Maestro's verify compares CODEX_HOME with the real profile and fails.
func (m *Manager) env(home string) []string {
	return []string{"ORQUESTRADOR_MAESTRO_SKIP_CLI_UPDATE=1", "ORQUESTRADOR_MAESTRO_TELEMETRY=0", "XDG_CONFIG_HOME=" + filepath.Join(home, "config"), "HOME=" + home, "USERPROFILE=" + home, "CODEX_HOME=" + filepath.Join(home, ".codex"), "NPM_CONFIG_USERCONFIG=" + filepath.Join(m.root, "npmrc"), "npm_config_cache=" + filepath.Join(m.root, "npm-cache"), "NPM_CONFIG_REGISTRY=https://registry.npmjs.org/", "NPM_CONFIG_UPDATE_NOTIFIER=false"}
}

// codexHome is the stable CODEX_HOME of Maestro agents (ADR 0014). Every verified release is activated into it.
func (m *Manager) codexHome() string {
	return filepath.Join(m.root, ".codex")
}

// stagingHome is where a release is installed and verified before it can become active.
func (m *Manager) stagingHome(version string) string {
	return filepath.Join(m.root, "versions", version, "home")
}

// activate installs and verifies one release's Codex target in the stable home. The home is shared by all
// releases, so it is written only after the release has verified in its staging home.
func (m *Manager) activate(ctx context.Context, version string) error {
	cli := m.cliPath(version)
	if _, err := m.runner.Run(ctx, m.root, m.env(m.root), cli, "install", "--home-path", m.root, "--only", "codex", "--non-interactive"); err != nil {
		return fmt.Errorf("install into Codex home: %w", err)
	}
	if _, err := m.runner.Run(ctx, m.root, m.env(m.root), cli, "verify", "--home-path", m.root); err != nil {
		return fmt.Errorf("verify Codex home: %w", err)
	}
	return nil
}

// restoreHome re-activates the release that stays selected after a failed switch.
func (m *Manager) restoreHome(ctx context.Context, version string) error {
	if version == "" {
		return nil
	}
	return m.activate(ctx, version)
}
func (m *Manager) cliPath(version string) string {
	name := "orquestrador-maestro"
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}
	return filepath.Join(m.root, "versions", version, "node_modules", ".bin", name)
}
func (m *Manager) currentVersion() (string, error) {
	b, err := os.ReadFile(filepath.Join(m.root, "current"))
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(b))
	major, _, _, stable := parseVersion(version)
	if !stable || major != 1 || compareVersions(version, pinnedLine) < 0 {
		return "", errors.New("maestro: current version pointer is invalid")
	}
	if _, err := os.Stat(filepath.Join(m.root, "versions", version, ".verified")); err != nil {
		return "", errors.New("maestro: current version is not verified")
	}
	return version, nil
}

func (m *Manager) previousVersion() (string, error) {
	b, err := os.ReadFile(filepath.Join(m.root, "previous"))
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(b))
	major, _, _, stable := parseVersion(version)
	if !stable || major != 1 || compareVersions(version, pinnedLine) < 0 {
		return "", errors.New("maestro: previous version pointer is invalid")
	}
	return version, nil
}

func (m *Manager) writeCurrent(version string) error {
	return m.writeVersionPointer("current", version)
}

func (m *Manager) writePrevious(version string) error {
	if version == "" {
		err := os.Remove(filepath.Join(m.root, "previous"))
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	major, _, _, stable := parseVersion(version)
	if !stable || major != 1 || compareVersions(version, pinnedLine) < 0 {
		return fmt.Errorf("%w: %q", ErrInvalidVersion, version)
	}
	return m.writeVersionPointer("previous", version)
}

func (m *Manager) writeVersionPointer(name, version string) error {
	if err := os.MkdirAll(m.root, 0o700); err != nil {
		return err
	}
	tmp := filepath.Join(m.root, name+".tmp")
	if err := os.WriteFile(tmp, []byte(version+"\n"), 0o600); err != nil {
		return fmt.Errorf("maestro: write %s version: %w", name, err)
	}
	if err := os.Rename(tmp, filepath.Join(m.root, name)); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("maestro: activate %s version: %w", name, err)
	}
	return nil
}

type preference struct {
	AutoUpdate bool `json:"auto_update"`
}

func (m *Manager) autoUpdate() (bool, error) {
	b, err := os.ReadFile(filepath.Join(m.root, "settings.json"))
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("maestro: read settings: %w", err)
	}
	var p preference
	if err := json.Unmarshal(b, &p); err != nil {
		return false, fmt.Errorf("maestro: decode settings: %w", err)
	}
	return p.AutoUpdate, nil
}
func (m *Manager) writeAutoUpdate(enabled bool) error {
	if err := os.MkdirAll(m.root, 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(preference{AutoUpdate: enabled})
	if err != nil {
		return err
	}
	tmp := filepath.Join(m.root, "settings.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(m.root, "settings.json")); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
func (m *Manager) cleanup(active, previous string) {
	entries, err := os.ReadDir(filepath.Join(m.root, "versions"))
	if err != nil {
		return
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := entry.Name()
		if _, err := os.Stat(filepath.Join(m.root, "versions", version, ".verified")); err != nil {
			if _, _, _, valid := parseVersion(version); valid {
				_ = os.RemoveAll(filepath.Join(m.root, "versions", version))
			}
			continue
		}
		if major, _, _, valid := parseVersion(version); valid && major == 1 {
			versions = append(versions, version)
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})

	retained := 0
	for _, version := range versions {
		if version == active || version == previous {
			retained++
		}
	}
	for _, version := range versions {
		if version == active || version == previous {
			continue
		}
		if retained < 3 {
			retained++
			continue
		}
		_ = os.RemoveAll(filepath.Join(m.root, "versions", version))
	}
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, dir string, env []string, executable string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, executable, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	envMap := make(map[string]string, len(os.Environ())+len(env))
	for _, value := range os.Environ() {
		key, val, ok := strings.Cut(value, "=")
		if ok && !controlledEnvironmentKey(key) {
			envMap[key] = val
		}
	}
	for _, value := range env {
		key, val, ok := strings.Cut(value, "=")
		if ok {
			envMap[key] = val
		}
	}
	cmd.Env = make([]string, 0, len(envMap))
	for key, value := range envMap {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	// Only stdout is returned: npm writes notices to stderr, and callers parse stdout as JSON.
	// Command output never reaches the error, since it can carry tokens or paths.
	cmd.Stderr = io.Discard
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", runnerOperation(executable, args), runnerFailure(err))
	}
	return out, nil
}

// runnerOperation names the command without its arguments beyond the subcommand.
func runnerOperation(executable string, args []string) string {
	name := strings.TrimSuffix(filepath.Base(executable), filepath.Ext(executable))
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name += " " + args[0]
	}
	return name
}

// runnerFailure keeps only the exit status of a failed command.
func runnerFailure(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Errorf("exit status %d", exitErr.ExitCode())
	}
	return err
}

type npmRegistry struct {
	runner Runner
	root   string
	env    []string
}

func (r npmRegistry) Versions(ctx context.Context) ([]string, error) {
	// The registry is queried before the first install, when root does not exist yet.
	if err := os.MkdirAll(r.root, 0o700); err != nil {
		return nil, fmt.Errorf("create maestro directory: %w", err)
	}
	out, err := r.runner.Run(ctx, r.root, r.env, "npm", "view", packageName+"@^"+pinnedLine, "version", "--json", "--registry=https://registry.npmjs.org/")
	if err != nil {
		return nil, err
	}
	var versions []string
	if err := json.Unmarshal(out, &versions); err == nil {
		return versions, nil
	}
	var version string
	if err := json.Unmarshal(out, &version); err == nil {
		return []string{version}, nil
	}
	return nil, errors.New("parse npm versions: invalid JSON response")
}
func parseVersion(v string) (int, int, int, bool) {
	if !versionPattern.MatchString(v) || strings.Contains(v, "-") {
		return 0, 0, 0, false
	}
	var a, b, c int
	if _, err := fmt.Sscanf(v, "%d.%d.%d", &a, &b, &c); err != nil {
		return 0, 0, 0, false
	}
	return a, b, c, true
}
func compareVersions(a, b string) int {
	am, an, ap, _ := parseVersion(a)
	bm, bn, bp, _ := parseVersion(b)
	if am != bm {
		if am < bm {
			return -1
		}
		return 1
	}
	if an != bn {
		if an < bn {
			return -1
		}
		return 1
	}
	if ap < bp {
		return -1
	}
	if ap > bp {
		return 1
	}
	return 0
}
func controlledEnvironmentKey(key string) bool {
	for _, controlled := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "CODEX_HOME", "ORQUESTRADOR_MAESTRO_SKIP_CLI_UPDATE", "ORQUESTRADOR_MAESTRO_TELEMETRY", "NPM_CONFIG_USERCONFIG", "NPM_CONFIG_CACHE", "NPM_CONFIG_REGISTRY", "NPM_CONFIG_UPDATE_NOTIFIER"} {
		if strings.EqualFold(key, controlled) {
			return true
		}
	}
	return false
}
