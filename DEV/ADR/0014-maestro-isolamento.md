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
- ADR 0013 stays as written for the harness default (shared `~/.codex`). The Codex harness runs with `CODEX_HOME` only in runs that use Maestro agents. Wiring that switch belongs to F1-T16 and is not built yet.
- Maestro's `rollback --to <stamp>` and `update` with `ORQUESTRADOR_MAESTRO_SKIP_CLI_UPDATE=1` operate on the Calango home only. Without that flag, `update` changes the global npm package, so it is never called (decision 4).
- Telemetry stays off without relying on the user's environment.

## Alternatives rejected
- **Install into the real `~/.codex` and other tool folders.** Writes into personal configuration and can overwrite the person's own agents.
- **No Codex target.** `verify` keeps failing, so Maestro cannot be activated.
- **Copy `auth.json` into the Maestro `.codex`.** Copies a credential file; rejected in ADR 0013.

## Open
- The implementation currently gives each release `<calango-home>/maestro/versions/<version>/home/.codex`, not the stable `<calango-home>/maestro/.codex` decided above. Automatic update can therefore switch to a signed-out profile. Resolve this implementation gap without copying the person's `~/.codex/auth.json`; no real Codex session has been tested.
- Verify on a real install that the agreed stable `CODEX_HOME=<calango-home>/maestro/.codex` loads installed agents and prompts, including skills; not run because no real Codex session is authorized.
