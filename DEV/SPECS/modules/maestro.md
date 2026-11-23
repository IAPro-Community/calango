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
- Automatic update: off by default; turned on by the owner. Never runs during a run.
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
