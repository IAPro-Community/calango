# ADR 0023 — Integrated terminal: PTY in the daemon, socket in the Electron main process

Status: proposed · 2026-11-25 · implements F4-T03 (issue #84) as an anticipated phase 4 foundation · builds on ADR 0015 (WebSocket) and ADR 0019 (desktop token handoff).

## Context

The ADE spec (`DEV/SPECS/modules/ade.md`) puts the terminal in `calango`: a PTY (ConPTY on Windows) streamed over `/v1/pty/{id}` and drawn with xterm.js in the desktop. A terminal is arbitrary command execution as the node's user, so it needs the same authentication as every other route, a restricted working folder, a bounded number of sessions, and certain cleanup of every process it started. The command policy (issue #89) and per-task worktrees (issue #83) come later and need hooks, not implementations.

## Decision

1. **Package `internal/pty`.** `Start(Options) (Process, error)` is the platform layer. `Manager` owns sessions in memory only (no table, no migration); they die with the daemon.
2. **Unix: `github.com/creack/pty` v1.1.24 (MIT).** The shell is a session leader with the PTY as controlling terminal. Ending a session sends SIGKILL to the process group, to every process still in the session and to every descendant (Linux: `/proc`; macOS: `kern.proc.all` plus `getsid`), in two passes. Background jobs, which interactive shells move to their own process groups, are covered. A process that calls `setsid` itself and is reparented escapes; cgroups would close that gap and are out of scope.
3. **Windows: ConPTY on `golang.org/x/sys/windows`**, which the module already depended on. Mature ConPTY wrappers (`UserExistsError/conpty`, `charmbracelet/x/conpty`, MIT) were considered; they wrap the same three calls and do not manage process trees. The shell starts suspended, is assigned to a job object with `KILL_ON_JOB_CLOSE`, then resumes, so no child can start outside the job and the tree dies even if the daemon crashes. Requires Windows 10 1809+. Compiled in CI (`GOOS=windows`); not yet run on Windows.
4. **Routes.** `POST /v1/pty`, `GET /v1/pty`, `GET /v1/pty/{id}` (WebSocket) and `DELETE /v1/pty/{id}`, all with write scope. Authentication is the bearer header of ADR 0015, checked before the upgrade. Sessions are bound to the creating device; another device gets 404. One client per session (409 for a second). Binary frames carry raw bytes; text frames carry JSON control (`resize` from the client, `exit` from the server).
5. **Limits.** At most 8 sessions per node (429). A session with no client attached and no input for 30 minutes is closed. Client frames are limited to 64 KiB. Session ids are 256 random bits.
6. **Working folder.** The shell starts only in the daemon workspace (`<data-dir>/workspace`) or below it, after symlinks are resolved. `pty.DirResolver` is the hook for per-task worktrees (#83).
7. **Program and environment.** The daemon chooses the shell (login shell from `$SHELL`, else `/bin/sh`; PowerShell, else `%ComSpec%`); clients cannot choose a program. The child environment is an allowlist (PATH, HOME, locale, temp, display and Windows system variables); `CALANGO_*`, provider keys and tokens never reach it.
8. **Command policy hook.** `pty.StartPolicy` can refuse a session before it starts. Sessions opened by a person pass. Issue #89 will plug the allowlist and approvals there for agent-driven terminals; typed keystrokes are not filtered.
9. **Audit.** `terminal.opened` and `terminal.closed` (actor: device; facts: session prefix, shell name, relative cwd, close reason, exit code). Terminal content is never recorded.
10. **Desktop.** The renderer uses the SDK for REST. The WebSocket runs in the Electron main process (`PtyBridge`) with the `Authorization` header (Node's WebSocket accepts headers), as ADR 0015 anticipated; the renderer exchanges bytes over IPC, checked by origin and argument shape. xterm.js 6.0.0 and `@xterm/addon-fit` 0.11.0 (MIT) draw the terminal.
11. **CSP change.** xterm.js inserts `<style>` elements for its viewport and theme, so the renderer CSP becomes `style-src 'self' 'unsafe-inline'`. `script-src` stays `'self'`. This relaxes ADR 0019 decision 5 for styles only.

## Consequences

- One new direct Go dependency (`creack/pty`); `golang.org/x/sys` moves from indirect to direct. Two new npm dependencies in the desktop.
- Shutting down the daemon (SIGINT/SIGTERM) kills every terminal tree before the HTTP server and the store close. A crash relies on the job object (Windows) or on the kernel's SIGHUP to the session (Unix); orphaned jobs that ignore SIGHUP can survive a Unix crash.
- Sessions do not survive a daemon restart. The desktop reattaches to a running session when the screen opens again.
- Not covered yet: Windows and macOS runtime tests, multiple simultaneous viewers, recording, the command policy itself.
