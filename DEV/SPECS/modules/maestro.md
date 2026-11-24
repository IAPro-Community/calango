# Module: maestro

Phase 1 (installation and linking) · Phase 3 (automatic update with rollback) · Package `internal/maestro` · ADR 0009 · Guide: [../ORQUESTRADOR_MAESTRO.md](../ORQUESTRADOR_MAESTRO.md)

## Goal
Embed the Orquestrador Maestro (V1 line, npm `@iapro/orquestrador-maestro-cli`) and follow its official updates without risk to the agents.

## Interface (Go)
```go
type MaestroManager struct { /* home, npm runner, registry client, store, clock */ }
func (m *MaestroManager) Status(ctx context.Context) (Status, error)
func (m *MaestroManager) Install(ctx context.Context, version string) error   // writes new version, verifies, switches pointer
func (m *MaestroManager) CheckUpdate(ctx context.Context) (*Release, error)  // queries the registry, V1 line only
func (m *MaestroManager) Rollback(ctx context.Context) error
```

## Rules
- Line pinned to `^1.1.1`. A major version is never installed on its own; always a card with a breaking-change warning.
- Installed under `<calango-home>/maestro/versions/<version>`; the `current` pointer points to the active one.
- `verify` is mandatory after installing or updating. If it fails → keep the previous one and record the reason.
- Daily check of the npm registry; a network failure does not raise an alert, it is only logged.
- Automatic update: on by default (D6); the owner can turn it off. Never runs during a run.
- An agent uses Maestro only if the owner linked it. Unlinking does not delete the installation.
- Cleanup: keep the last 3 versions that passed `verify`.
- Every version switch creates an `audit_log` entry with `from`, `to`, `auto` and the result.
- The "About" screen shows the active version, the Apache-2.0 license and a link to the repository.

## API
`GET /v1/maestro`, `POST /v1/maestro/install`, `POST /v1/maestro/update`, `POST /v1/maestro/rollback`, `PATCH /v1/maestro/settings` `{ auto_update: bool }`.

## Events
`maestro.update_available`, `maestro.updated`, `maestro.update_failed`.

## Acceptance
- A new V1 version with `auto_update` off generates a card and changes nothing.
- A version that fails `verify` does not replace the active one.
- Rollback restores the previous version and `verify` passes.
- Version 2.x is never installed without explicit confirmation.

## Probe findings (npm 1.1.1, 2026-11-23)

Probed in a temp directory with `--home-path` and a temp `HOME`. Not wired into Calango.

- `install`, `update`, `verify` and `rollback` accept `--home-path`. Core files go to `<home>/.orquestrador` and `<home>/AGENTS.md`; target files go to `<home>/.codex/...`. With a temp `HOME`, the only file written outside `--home-path` was `.config/orquestrador-maestro/telemetry.json`.
- Telemetry state is written to `$XDG_CONFIG_HOME/orquestrador-maestro` (or `$HOME/.config`) even with `--home-path`. Default is `enabled: false`, but an `anonymousId` is created. Calango must run the CLI with `ORQUESTRADOR_MAESTRO_TELEMETRY=0` and an `XDG_CONFIG_HOME` inside the Calango home.
- `verify` fails after a `--core-only` install (`Codex agents missing`, `Codex prompts missing`, exit 1). It passes after `--only codex`. Verification therefore needs at least the Codex target.
- JSON output exists only for `status --json`, `context brief|section --json`, `route explain --json` and `usage --json`. `install`, `update`, `verify`, `rollback` and `version` print text; `version --json` is rejected. `version --check` fails here with "npm retornou uma versão inválida": npm returns `["1.1.1"]` for `view <pkg>@latest version --json`, which the CLI does not parse. Calango must query the registry itself.
- `update` first runs `npm install -g <pkg>@<channel> --force --prefer-online` (global self-update), then re-runs itself with the new global CLI. Set `ORQUESTRADOR_MAESTRO_SKIP_CLI_UPDATE=1` to skip the global step. Without it, a `latest` channel could move the global package past the V1 line, which breaks the major-version rule. Calango should install pinned versions under `<calango-home>/maestro/versions/<v>` with npm `--prefix` and never call `update`.
- Applied non-interactively with the skip flag and a temp `HOME`: `update --only codex --non-interactive` exit 0, `verify` exit 0 afterwards. `rollback --to <stamp>` exit 0, `verify` exit 0 afterwards. `rollback --list` shows one backup per apply.
- Decided (ADR 0014): Maestro installs into `<calango-home>/maestro/`; its Codex target runs under its own `CODEX_HOME`, with a separate login. Not yet verified with a real Codex session.
- Validated again on 2026-11-23 with Calango's settings (`--home-path <calango-home>/maestro`, `ORQUESTRADOR_MAESTRO_TELEMETRY=0`, `XDG_CONFIG_HOME=<home>/config`, npm `--prefix versions/1.1.1`):
  - `install --home-path H --only codex --non-interactive` exit 0; `verify --home-path H` exit 0, `CoreOnly: false`. `verify` rejects `--only` ("Parametro desconhecido"): pass `--home-path` only.
  - No change under the real `$HOME` (marker check, npm cache excluded). Telemetry state is at `H/config/orquestrador-maestro/telemetry.json`.
  - `--only codex` also installs a Codex desktop plugin at `H/.codex/plugins/orquestrador-maestro` and asks the user to reopen the app and approve hooks. It stays inside the Calango home. Not verified whether Codex reads it with `CODEX_HOME`.
  - `ORQUESTRADOR_MAESTRO_SKIP_CLI_UPDATE=1 update --home-path H --only codex --non-interactive` exit 0; `verify` exit 0 afterwards.
  - `rollback --home-path H --list` exit 0, lists one stamp. `status --json --project-path H` exit 0, valid JSON (`jq -e`).

## Current implementation status (F1-T16, partial)

- `internal/maestro` installs verified V1 packages under per-version directories, exposes status/install/update/rollback/settings APIs, links agents, audits switches, checks for updates daily and defaults automatic updates on. Tests use fake npm/registry/Codex executables; no real Codex session was run.
- Current `CodexHome` is `<calango-home>/maestro/versions/<version>/home/.codex`. This differs from ADR 0014's stable `<calango-home>/maestro/.codex`; each release has a separate login, so auto-update can switch to a signed-out profile. No real user login is copied. Resolve the path/design gap before calling F1-T16 complete.
- The settings screen states the per-version profile behavior and requires separate sign-in; rollback is offered only when a previous verified version exists.
