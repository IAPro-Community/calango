# ADR 0014 — Maestro installs into an isolated Calango home

Status: accepted · 2026-11-23 · refines [ADR 0009](0009-orquestrador-maestro.md) (installation) and [ADR 0013](0013-harness-cli-como-provedor.md) (Codex home)

## Context
The Orquestrador Maestro npm CLI (`@iapro/orquestrador-maestro-cli` 1.1.1) installs by default into the person's home: `AGENTS.md`, `.orquestrador/` and, per target, `.codex/`, `.claude/` and others. A probe (see `SPECS/modules/maestro.md`, "Probe findings") showed:
- `--home-path <dir>` moves core and target files under `<dir>`, and nothing else was written outside it.
- Telemetry state is written to `$XDG_CONFIG_HOME/orquestrador-maestro` (or `$HOME/.config`) even with `--home-path`. Default is `enabled: false`, but an `anonymousId` is created.
- `verify` requires the Codex target (`.codex/agents`, `.codex/prompts`) under the home path. A core-only install fails verification.
- Codex reads its home from `CODEX_HOME`. An empty `CODEX_HOME` reports "Not logged in", so Codex does not fall back to `~/.codex` when the variable is set.

## Decision
1. Maestro is installed into `<calango-home>/maestro/` with `--home-path` set to that directory. The person's `~/.codex`, `~/.claude` and other tool folders are never written by Calango.
2. Calango runs the CLI with `ORQUESTRADOR_MAESTRO_TELEMETRY=0` and `XDG_CONFIG_HOME=<calango-home>/maestro/config`, so telemetry state stays in the Calango home.
3. The Codex target lives at `<calango-home>/maestro/.codex`. Where Calango runs Codex with these Maestro agents, it sets `CODEX_HOME` to that directory. That copy has its own login. Calango does not copy `auth.json` from `~/.codex` (ADR 0013 rejects copying credentials).
4. Calango never calls Maestro's `update`: it runs `npm install -g <pkg>@<channel> --force` (global self-update) and then re-runs itself. Pinned versions install under `<calango-home>/maestro/versions/<v>` with npm `--prefix`, and the Maestro binary runs from there. Install and rollback use `--only codex`; core-only installs are not valid for activation because `verify` fails.

## Consequences
- The person signs in to Codex twice when both the personal CLI and Maestro-with-Codex are used. This is the cost of isolation.
- ADR 0013 stays as written for the harness default (shared `~/.codex`). Runs of Maestro-linked agents use `CODEX_HOME` set to the stable Maestro home: `cmd/calango` passes `Manager.CodexHome` to the conversation service, which sets it for Codex runs.
- Maestro's `rollback --to <stamp>` and `update` with `ORQUESTRADOR_MAESTRO_SKIP_CLI_UPDATE=1` operate on the Calango home only. Without that flag, `update` changes the global npm package, so it is never called (decision 4).
- Telemetry stays off without relying on the user's environment.

## Alternatives rejected
- **Install into the real `~/.codex` and other tool folders.** Writes into personal configuration and can overwrite the person's own agents.
- **No Codex target.** `verify` keeps failing, so Maestro cannot be activated.
- **Copy `auth.json` into the Maestro `.codex`.** Copies a credential file; rejected in ADR 0013.

## Open
- Resolved in code (2026-11-23): the Codex home is `<calango-home>/maestro/.codex`. Each release is installed and verified in its staging home `versions/<v>/home`, then activated into the stable home (`activate`, `restoreHome` in `internal/maestro/manager.go`). Tests: `TestCodexHomeIsStableAcrossReleases`, `TestFailedReleaseVerificationNeverWritesActiveCodexHome`. Throwaway CLI probe (`/tmp/mtest`, not in the repo): reinstall and 1.1.0 → 1.1.1 → 1.1.0 over one home keep `.codex/auth.json`; `verify` exit 0 each time.
- Migration policy (owner decision pending): logins stored by earlier builds under `versions/<v>/home/.codex` are not moved and not copied. The person signs in to Codex again once.
- Open: no real Codex session has run against the stable home, so installed agents and skills loading under it are unverified. No real Codex session is authorized.
