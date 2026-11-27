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
- Import skills from Claude Code (`.claude/`) and Codex (`.codex/`). Foundation delivered early (F4-T07, issue #88): discovery and versioned import in `internal/skills`, see `SPECS/modules/skills.md` and ADR 0022. Project roots are configured by hand (`CALANGO_SKILL_DIRS`) until the `projects` table exists; imported skills are not given to runs yet.

## Acceptance
- Two tasks in the same project run in parallel in different worktrees and produce two PRs.
- The person can review the diff and the test results without relying on the agent's final message.
- A project on the VPS is editable from the desktop with the same editor and terminal.
