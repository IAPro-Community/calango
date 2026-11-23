# Handoff

Updated: 2026-11-23.

## Confirmed state

- Go 1.27.4 installed in `~/.local/opt/go` (link at `~/.local/bin/go`). Project minimum stays Go 1.22.
- Phase 0: configuration and CI for six builds implemented. `go test`, `go test -race`, `go vet`, `gofmt`, six cross builds and a local health check passed (details in `DEV/VERIFY.md`).
- F0-T07: tokens and i18n each generate their constants from a single source (`packages/tokens/tokens.json`, `packages/i18n/messages.json`); `--check` for both runs in CI and passed locally.
- F1-T01/T02: `internal/store` (modernc SQLite, WAL, immutable migrations with ledger) and migrations `0001`–`0006`; `calango` opens `calango.db` at startup.
- F1-T03 gateway and F1-T06 agents service (domain, store, node-wide pause, monthly usage): implemented and tested.
- F1-T07 (partial): conversations and runs service on a fake provider, migration 0003. Guide/steer and WebSocket reconnection pending; see `DEV/SPECS/modules/conversations.md`.
- F1-T08 (partial): approvals domain, migration 0004, store and service (digest-bound snapshot, decision, receipt, "always allow", 24 h expiry). Card and `waiting_approval` pending. HTTP for approvals is served (list and decide). See `DEV/SPECS/modules/approvals.md`.
- F1-T09 (partial): routines domain (cron, quiet window, retry backoff), migration 0005, store and service; no routine loop runs at startup. Pending: routine HTTP routes, client events, real executor (D3), manual run, `routine_finish`, event triggers. See `DEV/SPECS/modules/routines.md`.
- F1-T11 (partial): memory service, migration 0006, store CRUD and purge; budgeted injection into runs. HTTP served (`/v1/memories`, CRUD). Pending: `memory_save` tool (D3), `project` scope (removed from the V1 contract), purge schedule. See `DEV/SPECS/modules/memory.md`.
- F1-T13 (partial): `internal/api` serves `/healthz` (public), `POST /v1/pair` (public, rate-limited), and behind the bearer token: agents CRUD with `pause-all` and `resume-all`, memories CRUD, approvals list and decide, pairing codes, devices list and revocation, and `GET /v1/usage`. Scopes are enforced per route (`read`, `write`, `approve`). `api/openapi.yaml` lints clean with no warnings. Not served yet: `/v1/conversations*`, `/v1/runs/*`, `/v1/routines*`, WebSocket `/v1/events`. Those routes are in the contract but return 401 without token and 404 with it.
- F1-T14 (partial): `internal/auth` creates a 256-bit token at `DataDir/api-token` (mode 0600) on first start; the daemon refuses a file readable by others. Owner `owner` and device `local-desktop` are constants in `cmd/calango` until accounts and pairing exist. Vault storage (F1-T10) pending.
- F1-T15 (partial): device pairing. A one-time 8-character code (5 min, `POST /v1/pairing-codes`) is redeemed at `POST /v1/pair` for a 256-bit device token. Migration `0007_devices.sql`; only SHA-256 digests are stored; the scope comes from the code (`full`, `approve_only`, `read`); revocation takes effect on the next call. QR, mobile app and administrator password pending. See `DEV/SPECS/modules/pairing-auth.md`.
- F1-T17 (partial): `internal/usage` service and `GET /v1/usage` over the `usage` ledger, filtered by the owner's agents. Groups: agent (default), routine, project, node, model. `cost_micros` sums known and estimated rows; unknown rows add nothing and are counted in `cost_unknown_rows`. Not done: known/estimated split, the panel (desktop screens), and projects (no table in V1). Smoke-tested on the built binary.
- Smoke check of the binary passed: `/healthz` 200; `/v1/agents` 401 without token and with a wrong token; create 201 and list 200 with the token.
- Research and positioning: `DEV/COMPETITIVE_LANDSCAPE.md`. Technical delivery pipeline: `DEV/TECHNICAL_DELIVERY.md`. TODO: `DEV/TODO_V1.md`.
- Docker: build and health check history in `DEV/VERIFY.md`; not revalidated because the sandbox denied access to the daemon. Bridge networking test still pending.
- License AGPL-3.0 + commercial license (ADR 0010). CLA and contacts still need owner review before publication.
- Repository has no automatic commit; do not publish to `IAPro-Community/calango` without an explicit request. Nothing has been committed or pushed.
- Language policy: the repository is in English (see `DEV/CONVENTIONS.md`). README exists in English, Spanish and Brazilian Portuguese.

## Next technical step

1. F1-T13 remainder: conversations and runs routes (after the guide/steer decision), routine routes, `/v1/events` WebSocket (choose the dependency and check its license and Go 1.22 support first; no silent SSE substitute).
2. F1-T15 remainder: QR pairing and the mobile app (phase 2, Flutter). Before any VPS deployment, make the `/v1/pair` rate limit proxy-aware (it keys on `RemoteAddr` today).
3. F1-T10 vault, then move the token into it (F1-T14).
4. F1-T07: guide/steer and "one reply at a time" per agent (needs a harness, F1-T04/T05, blocked by D3).
5. F1-T08: card, `waiting_approval` on runs, reconciliation of approved actions without receipt.
6. F1-T09: real executor (D3), `POST /run`, `routine_finish`.
7. F1-T11: `memory_save` (D3), purge schedule.
8. Then F1-T12 (MCP), F1-T16 (Maestro via npm), F1-T17 (known/estimated split, panel), F1-T18 (audit), desktop Electron (F1-T20..26; `apps/desktop` does not exist).

## Owner decisions and publication limits

- D3 first harness (Claude Code CLI / Codex / direct API): blocks F1-T04/T05, guide/steer, real executor, `memory_save`, conversation responses. No answer yet.
- D1b: legal review of `CLA.md`, `COMMERCIAL.md`, `TRADEMARKS.md`, legal name and contacts.
- D2 Pro/Team screen control; D4 domain/module name; D5 `tsnet`; D6 automatic Maestro update.
- Run the workflow on GitHub after an authorized push; local lint and cross builds already passed.
- Full visual review of the site on phone and in dark mode; Docker bridge test on a compatible environment.
- Maestro: validate install destination, JSON format and non-interactive `update` per `DEV/ORQUESTRADOR_MAESTRO.md`.
- Owner items for the site: CLA legal name, contact emails (`licenca@`, `seguranca@`, `conduta@` at calango.dev), Caddy domain placeholder, Terms of Service link.
