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

## Current implementation status

- Preview (F4-T06, #87, foundation ahead of the F4 acceptance): the desktop detects local dev servers on loopback and opens them in the internal browser, in a preview profile separate from the agents and confined to the chosen origin. Details and limits in `browser.md` ("Project preview"). It does not depend on projects or worktrees yet; `openPreview(target, { projectId })` is the hook for #82/#83.

## Acceptance
- Two tasks in the same project run in parallel in different worktrees and produce two PRs.
- The person can review the diff and the test results without relying on the agent's final message.
- A project on the VPS is editable from the desktop with the same editor and terminal.
