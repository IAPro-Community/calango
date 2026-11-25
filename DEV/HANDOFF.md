# Handoff

Updated: 2026-11-25.

## Confirmed state

- Go 1.27.4 installed in `~/.local/opt/go` (link at `~/.local/bin/go`). Project minimum is Go 1.23.
- Phase 0: configuration and CI for six builds implemented. `go test`, `go test -race`, `go vet`, `gofmt`, six cross builds and a local health check passed (details in `DEV/VERIFY.md`).
- F0-T07: tokens and i18n each generate their constants from a single source (`packages/tokens/tokens.json`, `packages/i18n/messages.json`); `--check` for both runs in CI and passed locally.
- F1-T01/T02: SQLite store and immutable migrations `0001`–`0014`; `calango` opens its database at startup.
- F1-T03 gateway and F1-T06 agents service (domain, store, node-wide pause, monthly usage): implemented and tested.
- F1-T07 (done): persisted conversations/runs, provider chain, stop/steer/retry, crash recovery, per-agent one-active-run FIFO, durable replies and REST reconnect restoration. WebSocket replay is intentionally omitted by ADR 0015.
- F1-T08 (partial): canonical approval snapshots/digests, API decision, waiting state, receipt and startup reconciliation. API integration test exercises a service request and HTTP decision using a fake executor; renderer card dispatch is mocked. No live provider/connector action.
- F1-T09 (done, Q33): routines domain (cron, interval, quiet window, retry backoff), migrations `0005` and `0012`, store, service; executor fires one agent run per attempt in one DM per routine (ADR 0017); HTTP `/v1/routines*`; `routine.fired` and `routine.failed` events; 30 s tick on the PC node. Event triggers (phase 3) and known limits in `DEV/SPECS/modules/routines.md`.
- F1-T11 (partial): memory CRUD, budgeted injection and `memory_save` through MCP. Purge runs at startup and daily. Project scope is removed from V1; owner addendum for channel/agent action memory remains open.
- F1-T13: OpenAPI REST routes and owner-filtered WebSocket `/v1/events` are served and tested; WebSocket has no replay by ADR 0015. Approval service request + HTTP decision is integration-tested with a fake executor; no live Codex action.
- F1-T04 (done): direct Anthropic provider in `internal/anthropic` (vault key read per call; SSE streaming; exact tokens). Agents reach it through their provider chain (Q32, ADR 0016); the daemon resolves its accounts per owner.
- F1-T14 (done): the local API token is the vault item `api-token` (`internal/auth`), created on first start. A legacy `DataDir/api-token` file is imported into the vault and removed; a conflicting vault value refuses the start. Every `/v1` route requires it. `POST /v1/auth/rotate-token` replaces it at once; the old token gets 401 on the next call (Q30). There is no route that revokes the local token without replacing it. Owner `owner` and device `local-desktop` are constants in `cmd/calango` until accounts and pairing exist.
- Vault HTTP (F1-T10, Q29): `GET /v1/vault` (names and dates), `PUT /v1/vault/{name}` (set) and `DELETE /v1/vault/{name}`. Full scope only. No route returns a value. `api-token` is reserved: PUT and DELETE return 409; it changes only through `POST /v1/auth/rotate-token`. Tests in `internal/api/vault_test.go`.
- Startup change (ADR 0012): the daemon refuses to start without a master key. Set `CALANGO_VAULT_KEY_FILE` (or `CALANGO_VAULT_KEY`) or unlock the OS keyring. Owner acknowledged (2026-11-23).
- Master-key rotation at startup (Q31, ADR 0012 amendment): `CALANGO_VAULT_KEY_VERSION`, `CALANGO_VAULT_KEY_PREVIOUS[_FILE]`; `Service.Rotate` runs before serving and audits each re-wrap as `vault.rewrap`; startup refuses rows whose version has no configured key. Smoke-tested on the built binary (`DEV/VERIFY.md`, 2026-11-23).
- F1-T15 (partial): device pairing. A one-time 8-character code (5 min, `POST /v1/pairing-codes`) is redeemed at `POST /v1/pair` for a 256-bit device token. Migration `0007_devices.sql`; only SHA-256 digests are stored; the scope comes from the code (`full`, `approve_only`, `read`); revocation takes effect on the next call. QR, mobile app and administrator password pending. See `DEV/SPECS/modules/pairing-auth.md`.
- F1-T17 (partial): REST and desktop usage panel exist; fake-run E2E shows token counts and an unknown-cost row. No verified Codex price or real provider cost yet; project group has no V1 projects table.
- F1-T18: audit hash chain, verify/export routes and desktop screen are implemented. Isolated UI smoke verified the chain and downloaded JSONL/CSV. Pre-0008 rows still verify as breaks from the first row, accepted pre-release.
- Smoke check of the binary passed: `/healthz` 200; `/v1/agents` 401 without token and with a wrong token; create 201 and list 200 with the token.
- Desktop fake-provider E2E passes agent creation, pause/resume, visible chat reply, routine create/toggle and usage. Settings/audit UI and Linux startup-at-login were exercised in an isolated Electron profile; no real model, production data or actual provider cost was used.
- The desktop was restarted with the same isolated profile/data directory already in use; `/healthz` on `127.0.0.1:7317` returned 200, CDP disabled. The default `~/.config/calango` database was not opened or modified.
- Research and positioning: `DEV/COMPETITIVE_LANDSCAPE.md`. Technical delivery pipeline: `DEV/TECHNICAL_DELIVERY.md`. TODO: `DEV/TODO_V1.md`.
- Docker: site and node images built and smoke-tested on a user-defined bridge. Both `/healthz` endpoints passed; `/data/calango.db` persisted through container recreation in a named volume. Caddy/TLS Compose and public VPS deployment remain unverified.
- License AGPL-3.0 + commercial license (ADR 0010). CLA and contacts still need owner review before publication.
- Publication: `main` pushed to `origin` on the owner's request on 2026-11-23 (`git log origin/main -1` gives the pushed head). The earlier note that the owner chose not to commit is superseded. `/tmp/agenticos` is kept until P1–P3 are done (owner, 2026-11-23).
- AgenticOS port (ADR 0011): `third_party/agenticos/LICENSE` and `NOTICE` copied byte for byte; root `NOTICE` has the entry; port checklist is item 10 of ADR 0011. ADR 0011 is accepted by the owner (Q01); legal review (D1b) is still open.
- Codex harness (F1-T05 partial, ADR 0013 and ADR 0016, owner 2026-11-23): `internal/harness.Codex` is a gateway provider named `codex`, tested with a fake `codex` binary only. Sandbox `workspace-write` (owner accepted the risk): Codex can write files and run commands without Calango's approval until MCP (F1-T12) routes its tools through the gateway. Isolation: a non-default account runs with its own `CODEX_HOME` (`<data-dir>/accounts/codex/<label>`); `default` shares `~/.codex`; both run with `--ignore-user-config --ignore-rules --ephemeral` (no auth copy). Open: `--json` schema (usage is estimated; reply is one chunk), whether `~/.codex/AGENTS.md` still loads, and one real run (owner did not authorise a real model call yet). Wired into the daemon at startup (`cmd/calango`: binary `codex` on PATH, workdir `<data-dir>/workspace`); no price table, so cost stays unknown.
- Desktop run uncovered npm/npx `node_modules/.bin` shadowing `codex` and causing `harness_signed_out`; Electron now filters those entries from the daemon child's `PATH`. Fake CLI resolution and the full smoke now pass.
- Language policy: the repository is in English (see `DEV/CONVENTIONS.md`). README exists in English, Spanish and Brazilian Portuguese.

## Next technical step

1. Keep Phase 2–6 gated. Phase 1 exit requires Windows installation, a real approved action, a daily routine and an evidenced cost row.
2. F1-T16: the stable Codex home (ADR 0014) is in code and fake-tested. Remaining: the owner's migration policy for earlier per-version logins (not moved; the person signs in again) and one real Codex session on the stable home. Do not copy the user's personal Codex credentials.
3. Obtain owner authorization for a real Codex/MCP run and an external approval action; configure verified provider pricing only with authorization.
4. Complete platform acceptance: Windows installer/signing and Windows/macOS keyring/login runtime. Linux startup toggle is verified; OS-level shortcut activation is not.
- Owner blockers: D1b legal review, D5 written Tailscale clearance or alternate transport, site terms/prices/domain/copy, permission before deleting `/tmp/agenticos`, memory capture/retention design, and authorization for live provider/action checks.
- **Platform blockers:** Windows/macOS keyring; F1-T24 OS shortcut/tray and platform login; F1-T25 signed installer/update and platform verification; OS shortcut activation. GitHub run 45188079827 passed on `dee8efa`; it does not cover current changes.
- Remaining implementable engineering: Codex JSONL/other V1 harnesses, MCP connector credentials/built-ins/result durability, previous-binary and OS-keyring backup/restore acceptance, personal-data export/deletion, user docs, accessibility, reference-code review and site icon. The offline runbook and migration restore test exist.
- Phase 1 exit remains unmet; do not report V1 complete.
- Resume steps, checks and launch commands: `DEV/TODO_V1.md`, section "Status and resume point (2026-11-23)".

## Owner decisions and publication limits

- D3 first harness: answered (Codex CLI first, 2026-11-23). Other V1 harnesses: Claude Code CLI, OpenCode, Antigravity (`agy`), direct API.
- D1b: legal review of `CLA.md`, `COMMERCIAL.md`, `TRADEMARKS.md`, legal name and contacts.
- D2 Pro, D3 Codex CLI first, D4 keep `calango`, D6 Maestro auto-update on are answered. D5 research is complete: BSD-3-Clause source license; official plan limits and terms are in ADR 0005. The commercial embedding question is unresolved; owner must obtain written clearance or choose another transport.
- Run the workflow on GitHub after an authorized push; local lint and cross builds already passed.
- Site visual review is complete at desktop 1440 px and mobile 390 px in light/dark; no horizontal overflow at 390 px. Pricing, legal contacts/domain and future phone/cloud/team plan claims still need owner review. Docker bridge test remains blocked by daemon access.
- Maestro CLI behavior was validated in a throwaway home, and Calango manager/routes/audit/update/rollback now exist. F1-T16 linked agents run with the stable `CODEX_HOME` (ADR 0014); no real Codex session was run.
- MCP (F1-T12) done in code (2026-11-23, ADR 0018): go-sdk v1.3.0 with Go minimum 1.23 (owner choice). Connectors, `read`/`write` grants, per-turn gateway at `/mcp`, writes through approvals, Codex wiring. Not verified live with Codex. Later go-sdk versions need Go 1.24/1.25: do not upgrade without a new owner decision.
- Owner items for the site: CLA legal name, contact emails (`licenca@`, `seguranca@`, `conduta@` at calango.dev), Caddy domain placeholder, Terms of Service link.
