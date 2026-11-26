# Module: ade (agent-driven development environment)

Phase 4 · Packages `internal/ade`, `internal/pty` · Tables `projects`, `worktrees`, `tasks`

## Goal
Work on separate projects with agents, on the PC or on the VPS.

## Rules
- Project = git repository + node + agents + connectors + memory (the project's `DEV/`, if it exists).
- Each task opens a **worktree** on its own branch. Several agents in parallel without conflicts.
- Terminal: PTY in `calango` (`creack/pty`; ConPTY on Windows), streamed over WebSocket (`/v1/pty/{id}`); xterm.js in the desktop app.
- Monaco editor, file tree, diff per worktree.
- Shell commands go through the safe list; anything outside it requires approval.
- Preview server (detected port) opens in the internal browser.
- GitHub/GitLab: create branch, open PR, read CI status.
- When the task ends: offer PR, keep or discard the worktree.
- Each run leaves an evidence package: diff, commands and exit codes, tests executed and artifacts produced. Failures and tests not run appear as such.
- The agent does not merge or publish/deploy without human approval tied to the exact action.
- Import skills from Claude Code (`.claude/`) and Codex (`.codex/`).

## Implemented: terminal (F4-T03, anticipated phase 4 foundation)

Delivered ahead of the phase 4 gate as a foundation (issue #84, ADR 0022). Phase 4 itself (projects, worktrees, editor, diff, PR) is not started.

- `internal/pty`: `Start` runs a program on a PTY (`creack/pty` on Unix; ConPTY through `golang.org/x/sys/windows` on Windows, inside a job object with `KILL_ON_JOB_CLOSE`). `Manager` keeps sessions in memory, at most 8 per node, closes a session with no client and no input after 30 minutes, and kills the whole process tree when a session ends or the daemon stops.
- Routes (write scope, bearer header as `/v1/events`): `POST /v1/pty` (body `{cwd, cols, rows}`, all optional), `GET /v1/pty`, `GET /v1/pty/{id}` (WebSocket: binary = raw bytes both ways; text = `{"type":"resize"}` from the client, `{"type":"exit"}` from the server), `DELETE /v1/pty/{id}`. Sessions are bound to the creating device. One client per session; 64 KiB of scrollback is replayed on attach.
- `cwd` must resolve, after symlinks, inside `<data-dir>/workspace`. The daemon picks the shell. The environment is an allowlist: no `CALANGO_*`, provider key or token.
- Audit: `terminal.opened`, `terminal.closed`, without terminal content.
- Hooks: `pty.DirResolver` for per-task worktrees (#83); `pty.StartPolicy` for the command policy (#89). Neither policy is implemented.
- Desktop: Terminal screen with xterm.js 6 and the fit addon. The WebSocket runs in the Electron main process with the bearer header; the renderer exchanges bytes over IPC. Renderer CSP allows inline styles for xterm.js.
- Not verified: Windows and macOS at runtime (both compile).

## Acceptance
- Two tasks in the same project run in parallel in different worktrees and produce two PRs.
- The person can review the diff and the test results without relying on the agent's final message.
- A project on the VPS is editable from the desktop with the same editor and terminal.
