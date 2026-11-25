# Worklog

## 2026-11-22 — foundation (phase 0)

- Reference scan: Dot (closed), Grok Bot, agentes.chat. Read README, CHANGELOG, ADRs, open issues and PRs. Clones stay outside the project, as reference only.
- Decided: core in Go, one `calango` binary; pure-Go SQLite; Wails for desktop; native mobile consuming the API (Flutter vs React Native still open at the time).
- Created skeleton: `go.mod`, `internal/core` (agent with budget, routine scheduler), `internal/api` (`/healthz`), `cmd/calangod`.
- Tests written, not run: Go was not installed on this machine.
- Decided: node-to-node transport via Tailscale in the initial phase (evaluate tsnet for embedding). Own relay later. Recorded in `DEV/PRODUCT_HYBRID.md`.
- 2026-11-23: full plan written in `DEV/PLANO.md` (vision, topology, stack, 17 modules, security, data, monetization, distribution, monorepo, 8-phase roadmap, lessons from references).
- 2026-11-23: desktop becomes Electron (previously Wails). Binary renamed from `calangod` to `calango` (`cmd/calango`), in code, Docker and docs.
- 2026-11-23: mobile decided as Flutter. Plan adjusted: OpenAPI generates TypeScript and Dart SDKs; tokens and translations with a single source; monorepo gains `api/`, `sdk-dart/` and `tokens/`.
- 2026-11-23: complete technical documentation so another agent can build it: `AGENTS.md` at the root, `DEV/INDEX.md`, `CONTEXT.md`, `CONVENTIONS.md`, `SECURITY.md`, `DATA_MODEL.md`, 8 ADRs, `SPECS/ROADMAP.md` with numbered tasks (F0–F7), 20 module specs in `SPECS/modules/`, `api/openapi.yaml` (draft v0.1), `packages/tokens/tokens.json` (palette sampled from the logo), `DEV/BRAND/BRAND.md`.
- 2026-11-23: presentation site in `site/index.html` (static): hero with a sticky-note wall and glass tile, product demo driven by a single clock (5 scenes, pause/drag/skip), animated PC↔VPS↔phone diagram, features, ADE with code and typing terminal, security, plans, platforms, roadmap, FAQ. Light/dark and `prefers-reduced-motion`. Logo copied to `DEV/BRAND/logo-original.png` and resized into `site/assets/`.
- 2026-11-23: Go 1.27.4 installed without sudo in `~/.local/opt/go` (`~/.local/bin/go`), SHA-256 checked against go.dev. `go vet`, `go test` (4 tests) and `gofmt` pass. `calango` starts and `/healthz` returns 200.
- 2026-11-23: Orquestrador Maestro (Apache-2.0) incorporated by ADR 0009: consumed from the npm package `@iapro/orquestrador-maestro-cli` on the V1 line (1.1.1), update by card or automatic (default off), `verify` and rollback. Documented in `DEV/ORQUESTRADOR_MAESTRO.md`, `SPECS/modules/maestro.md`, `NOTICE`. ADR 0006 gained an exception only for Maestro.
- 2026-11-23: `DEV/TODO_V1.md` created with all V1 tasks (roadmap phases 0–6 + Maestro + cross-cutting) and decisions D1–D6 that block parts of the delivery.
- 2026-11-23: owner decided that Calango code will be open source (D1). The specific license was still to be chosen (D1a); recommendation: Apache-2.0. No `LICENSE` had been created yet.
- 2026-11-23: license decided: AGPL-3.0 + commercial license + CLA + trademark policy (ADR 0010). Created `LICENSE` (official text, gnu.org), `COMMERCIAL.md`, `CLA.md` (draft for review), `TRADEMARKS.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `.gitignore`, `.dockerignore`, `.github/workflows/ci.yml`.
- 2026-11-22: review of the site (`site/index.html`). Contrast measured in the browser: green titles and eyebrows, primary button, roadmap numbers, ADE cards and tags in dark mode were below AA. Fixed with text tokens (`--moss-text`, `--btn`, `--num`, `--warn-text`, `--on-tone`). Serious bug fixed: the ADE section title was invisible in dark theme. Copy trimmed (leads, features from 12 to 8, security from 6 to 4 cards). New effects: capabilities tape, progress bar, marker drawn on titles, stage tilt with the mouse, tilt and hover on cards, all under `prefers-reduced-motion`.
- 2026-11-22: attempt to validate Docker blocked by the `docker` group. After the owner added the user to the group, both builds passed (site and node). Health checks of the site and node responded 200 with `--network host`. Bridge networking does not work on this machine (`operation not supported`).
- 2026-11-23: root Dockerfile became the **site** (nginx non-root, port 8080, healthcheck, CSP). Node Dockerfile moved to `deploy/Dockerfile.node`. "Contribute to V1" section added to the site.
- 2026-11-23: local git initialized on `main`, **no commit**. Nothing published.
- Deployment model: server on the person's VPS via Docker, clients connect over TLS. Documented in `DEV/DEPLOYMENT.md`. Dockerfile, compose and Caddyfile created, not tested.

## 2026-11-23 — harden the foundation and consolidate V1

- `internal/config` centralizes `CALANGO_ADDR`, keeps loopback by default and validates host/port before the scheduler; invalid configuration test added.
- `core.Agent.Charge` avoids `int64` overflow; regression test guarantees that a refused charge does not change usage.
- CI includes cross builds for Linux/Windows/macOS on amd64/arm64; the six builds, tests, race, vet and format passed locally. `GET /healthz` returned 200. The workflow still depends on a run on GitHub after an authorized push.
- Research with official sources in `DEV/COMPETITIVE_LANDSCAPE.md`; plan repositioning and approval, idempotency, unknown cost and evidence criteria adjusted in the specs and the V1 TODO.
- OpenAPI 3.1 lint added to CI with Redocly CLI MIT 2.81.0 pinned; operationIds and tag descriptions completed. Local lint passed with no warnings.
- `DEV/TECHNICAL_DELIVERY.md` created for the CI/release/migration/rollback flow. Docker was not revalidated: the sandbox did not allow access to the daemon.
- Deterministic token generator creates CSS and TypeScript/Dart constants; CI checks drift. `node scripts/generate-tokens.mjs`, `--check` and `node --check` passed. i18n was split out to F1-T26, together with the catalog creation.
- Next: F1-T01 (SQLite store) and F1-T02 (initial migrations); they do not depend on D3, which blocks only the provider/harness.
- 2026-11-23: repository language policy set to English. Contribution rules updated in `CONTRIBUTING.md`, `AGENTS.md` and `DEV/CONVENTIONS.md`. Public docs (`README` in English, Spanish and pt-BR, `CLA`, `COMMERCIAL`, `TRADEMARKS`, `SECURITY`, `CODE_OF_CONDUCT`, `NOTICE`) translated. Translation of `DEV/` in progress.
- 2026-11-23: translation of `DEV/` finished (ADRs, module specs, specs, architecture, security, deployment, data model, monetization, brand, competitive landscape). Filenames with Portuguese slugs kept so links keep working.
- 2026-11-23: site `site/index.html` now ships in English (default), Brazilian Portuguese and Spanish. Detection order: `?lang=` > `localStorage` `calango-lang` > `navigator.languages` (pt/en/es) > `en`. Language switcher in the nav; `<html lang>`, title, meta description and hreflang alternates follow the locale. Also fixed the rotating hero word clipping descenders.
- 2026-11-23: F1-T01/T02 done. `internal/store`: SQLite via `modernc.org/sqlite` v1.35.0 (last release that accepts Go 1.22), WAL and foreign-key pragmas, database file 0600 in a 0700 directory, immutable migration runner with `schema_migrations` ledger (sha256 checksum, one transaction per migration, rollback on failure). Initial migration `0001_initial.sql`: agents, conversations, messages, runs, run_steps, usage, audit_log (append-only by trigger). `CALANGO_DATA_DIR` added to `internal/config` (validated, absolute path); `cmd/calango` opens `calango.db` at startup. Smoke test passed. The rest of the DATA_MODEL goes in later migrations.
- Next: F1-T03 `internal/gateway`.
- 2026-11-23: F1-T03 done. `internal/gateway`: the `Provider` interface, `Gateway.Call` checks the agent's monthly budget and reserves the estimate (prompt/4 + max output) before the provider is contacted, and records real tokens after the call, including when the call fails (estimate charged) and when the context was cancelled. Cost is exact integer micros (`big.Int`, rounded up); a model with no price entry is `cost_status=unknown` with NULL cost, never zero. 80% and 100% thresholds are emitted once per crossing. `internal/store/usage.go` is the SQLite ledger; `internal/core/id.go` makes ULID-style ids for `usage.id`. Price table is empty for now: no verified prices in the repo. Next: F1-T06 agents CRUD.
- 2026-11-23: F1-T06 done (service and store; HTTP later). Domain `core.Agent` with `Normalize` (defaults `run_on=pc`, `approval_level=sensitive`) and `Validate` (name, `token_limit_month > 0`, enums); domain errors `core.ErrNotFound`, `ErrConflict`, `ErrInvalidInput`. `internal/agents` service: create, get, list, replace (PUT, full replacement), delete, node-wide pause and resume; views carry monthly usage in UTC. Store: `internal/store/agents.go`, migration `0002_node_state.sql` (singleton `node_state` for the pause switch). Delete is refused with conflict when runs or usage exist (FK RESTRICT). Contract: `PATCH` replaced by `PUT`, `DELETE` and `POST /v1/agents/resume-all` added, `AgentInput` defaults documented. Old `core.Charge`/`ErrBudgetExceeded` removed (no production use; budgets live in `internal/gateway`). Next: F1-T07.
- 2026-11-23: F1-T07 partial. `internal/conversations` service: DM, message and run admission in one transaction, parallel limit with queue, stop, retry with instructions (`retry_of`), paused node recorded as `failed/agent_paused`, crash recovery in `Start`. Migration `0003_conversation_members_and_retry.sql` (`conversation_members`, `runs.instructions`, `runs.retry_of`, status index); persistence in `internal/store/conversations.go`; types in `internal/core/conversation.go`. The owner-test budget used a limit of 1 for the budget case only: a 1000-token monthly limit is smaller than the 4096-token output reservation, so the gateway correctly refuses the call. Not done: guide (steer), per-agent "one reply at a time", WebSocket reconnection. F1-T07 stays unchecked. Next: steer when F1-T04/T05 exist, otherwise F1-T08 approvals.
- 2026-11-23: F1-T08 partial. `internal/core/approval.go`: `Action` with `Normalize` (sensitive kinds only, target trimmed, args as canonical JSON object) and `Digest` (SHA-256 of the canonical payload). Migration `0004_approvals.sql`: snapshot columns immutable, status and outcome change once (triggers), `approval_rules`. Persistence in `internal/store/approvals.go`. Service `internal/approvals`: request, decide (device required), execute only the approved snapshot after a digest check (mismatch → `approval.blocked` and a new request, `ErrDigestMismatch`), separate receipt, "always allow" rules, 24 h expiry. Fixed a flaky retry test in conversations (it stopped the run before the provider call). Not done: HTTP and card (F1-T13), `waiting_approval`, reconciliation after a crash.
- 2026-11-23: F1-T09 partial. Routines: `internal/core/cron.go` (5-field cron: `*`, ranges, steps, lists, day-of-week 7 = Sunday, day-of-month OR day-of-week when both restricted, local time zone injected), `internal/core/routine.go` (validation, `NextAfter`, quiet window that may cross midnight, retry backoff 1m/5m/30m, deterministic `OccurrenceID`). Migration `0005_routines.sql` (`routines`, `routine_runs` with `UNIQUE(routine_id, occurrence_id)`). Store in `internal/store/routines.go`. Service `internal/routines`: a due routine records one occurrence at its earliest missed time and moves `next_run_at` from now (missed times coalesce); quiet window postpones without counting an attempt; a failed attempt retries the same row with backoff until `max_retries`, then notifies the owner once; `notify_on` applies to success; `Start` requeues occurrences left running by a crash; `run_on` routes to this node (`pc` or `vps`). Removed the in-memory `core.Scheduler` (dead: no routine was ever registered, and its `Routine` name collided with the domain type); `cmd/calango` no longer starts a scheduler loop. Not done: HTTP routes, client events, real executor (needs a harness, D3), manual run, `routine_finish`, event triggers.
- 2026-11-23: F1-T11 partial. Memory: `internal/core/memory.go` (scope `account`/`agent`, kind `fact`/`preference`/`decision`, text up to 2000 characters, `EstimateTokens` = runes/4). Migration `0006_memories.sql` (CHECK that `agent_id` is set only for agent scope; cascade on agent delete). Store in `internal/store/memories.go` (insert, get, list, update, logical delete, `PurgeMemories`). Service `internal/memory`: create, edit, delete (immediate, invisible to other owners), purge after 30 days as a method (no scheduler), and `Context`, which builds a block of up to 1500 tokens with the most recent memories and a header marking it as context, not instructions. `internal/conversations` injects the block into the prompt only when there are memories (optional `Config.Memory`). Not done: `memory_save` tool (needs a harness, D3), HTTP routes, `project` scope (no `projects` table), purge loop, semantic relevance (V1 ranks by recency).
- 2026-11-23: F1-T13 partial (REST for agents, memories and approvals). `api/openapi.yaml`: memory scope `project` removed (no projects table in V1); `Error.code` gained `internal`; `POST /v1/memories` (`createMemory`, `MemoryInput`), `PATCH /v1/memories/{id}` and `DELETE /v1/memories/{id}` added. `internal/api`: `NewHandler(Config)` validates services, token, owner and device; `/healthz` is public and every other route sits behind `requireToken` (constant-time bearer compare, 401 with `WWW-Authenticate`). Errors use `{error:{code,message}}` (`not_found`, `conflict`, `invalid_input`, `unauthorized`, `internal`; internal details are logged, not returned). Bodies are capped at 1 MiB; unknown fields and trailing data are rejected. Served: agents CRUD with `pause-all` and `resume-all`, memories CRUD, approvals list and decide. Contract-only, not served: `/v1/conversations*`, `/v1/runs/*`, `/v1/routines*`, `/v1/usage`, `/v1/pair`, WebSocket `/v1/events`. `internal/auth` (F1-T14 partial): `LoadOrCreateToken` writes a 256-bit token to `DataDir/api-token` (mode 0600) and refuses a file readable by others or empty. `cmd/calango` wires the services with owner `owner` and device `local-desktop`; both are constants until accounts and pairing exist (F1-T15).

## 2026-11-23 — device pairing (F1-T15, partial)

- Migration `0007_devices.sql`: `pairing_codes` (digest, owner, scope, expiry, use) and `devices` (digest unique, platform `desktop|ios|android|web`, scope, last access, revocation). Domain in `internal/core/device.go`; token helpers in `internal/auth/device.go`; persistence in `internal/store/devices.go`; service in `internal/devices` (issue code, pair, authenticate, list, revoke; audit actor kind `device`).
- Pairing redeems a code with one conditional UPDATE, so a code is used once even under concurrent requests. Name and platform are validated before the code is consumed, so a typo does not burn it.
- HTTP: `POST /v1/pair` (public, 5 attempts per minute per `RemoteAddr`, then 429), `POST /v1/pairing-codes`, `GET /v1/devices`, `DELETE /v1/devices/{id}`. Scopes enforced per route: agent and memory reads need `read`, their writes and device routes need `full`, approvals need `approve`.
- Local desktop token remains a full-scope credential outside the device table; `cmd/calango` wires the devices service.
- Contract: `api/openapi.yaml` gained the four routes and `forbidden`; the store schema test now lists `devices` and `pairing_codes`.
- Next: QR pairing and mobile app (phase 2); proxy-aware rate limit before a VPS deployment.

## 2026-11-23 — usage endpoint (F1-T17 partial, F1-T13 partial)

- `api/openapi.yaml`: `GET /v1/usage` documented (`from` inclusive, `to` exclusive, `group_by` default `agent`); `UsageRow` requires `cost_unknown_rows`; `cost_micros` documented as known plus estimated.
- `internal/usage`: `Query`, `Row`, `Group` (agent, routine, project, node, model), `Service.Summary` validating group and range before reading. Repository implemented by `store.UsageTotals`, which joins `usage` to `agents` for the owner and takes the grouping column from a fixed map (no interpolated input).
- `internal/api/usage.go`: route behind `PermRead`; `from`/`to` parsed as RFC 3339; invalid input 400.
- Wired in `cmd/calango/main.go` and `internal/api/api_test.go`.
- Not done: known/estimated split, panel, projects table. Next: F1-T10 vault, then WebSocket and conversation/run routes once D3 is answered.

## 2026-11-23 — site header fixes

- `site/index.html` nav: 24 px gap between brand, menu and CTA; links `nowrap`; compact links and gaps at ≤1100 px; CTA label on one line; brand and CTA `flex-shrink:0` (the brand shrank at 1001 px and its text overlapped the menu).
- Menu hidden at ≤1020 px (was ≤860 px, which let Spanish overflow between 900 and 1000 px). No mobile menu was built; it is a future decision.
- ≤520 px: 16 px side padding and smaller language and theme controls, so 320 px fits.
- Plan grid at ≤920 px uses `minmax(0,1fr)`: the cards overflowed the viewport at 320 px in Spanish.
- Language detection normalizes `?lang=`, `localStorage` and `navigator.languages` to the base code: `?lang=pt-BR` now selects PT.
- Kept the two-letter "PT" label; "PT-BR" widens the switcher and reopens the 320 px collision.

## 2026-11-23 — AgenticOS reuse (ADR 0011) and audit chain (F1-T18 partial)

- Decided ADR 0011: Apache-2.0 code from `vstorm-co/agenticos` at commit a4b0b3c may be ported to Go, TypeScript and Dart with per-file attribution. Not ported: prompts, brand marks, Tauri, sandbox, channels, voice, workflows, Vstorm deploy and the Python runtime.
- Removed a third-party reference name from all documentation (ADR 0006 applies). Older git history still contains it.
- Created `third_party/agenticos/LICENSE` and `NOTICE` (byte-for-byte copies) and the AgenticOS entry in the root `NOTICE`. Port checklist added as item 10 of ADR 0011.
- F1-T18 audit chain, first port (from `core/audit.py` and `services/audit.py`): migration `0008_audit_chain.sql`; `internal/audit` (canonical SHA-256 hash over prev hash and fields, `Verify`); `internal/store/audit.go` (`insertAudit` moved out of `approvals.go`, `VerifyAudit`). The hash excludes row id and sequence number.
- The schema contract test in `internal/store/store_test.go` lists `audit_checkpoints`.
- Documentation aligned: AGENTS.md exception, TODO_V1 and ROADMAP link P1–P3 items to ADR 0011, module specs carry source references, F1-T18 marked partial.
- Not done: HTTP verify route, export, UI. `modules/usage.md` does not exist; the F1-T17 source is recorded in TODO_V1 and ROADMAP.

## 2026-11-23 — Vault storage and API token (F1-T10 partial, F1-T14 partial, Q12–Q14)

- Licence check for the OS keyring (Q12): `github.com/zalando/go-keyring` v0.2.8 (MIT) with `github.com/godbus/dbus/v5` (BSD-2-Clause) and `github.com/danieljoos/wincred` (MIT); `golang.org/x/sys` (BSD-3-Clause). Listed in root `NOTICE`.
- ADR 0012 (envelope): values in migration `0009_vault_items.sql`, AES-256-GCM, data key wrapped with HKDF-SHA256 bound to owner and key version; `Rewrap` for rotation. Code: `internal/vault` (envelope, keys, service, bootstrap) and `internal/store/vault.go`.
- Master key sources: OS keyring, or `CALANGO_VAULT_KEY` / `CALANGO_VAULT_KEY_FILE` (mutually exclusive). Both are read only in `internal/config`, not in `main`, per `DEV/SPECS/modules/config.md`. No plain-text fallback.
- API token moved to the vault as `api-token` (`internal/auth`). A legacy `DataDir/api-token` is imported and removed; a conflicting vault value refuses the start and neither value is deleted.
- Startup change: the daemon refuses to start without a master key. Linux without Secret Service needs `CALANGO_VAULT_KEY_FILE` (or an unlocked keyring) before the first start. Owner acknowledgement pending.
- `cmd/calango/main.go` wires the vault and the token import; the listen log no longer names a file.
- Not done: vault audit events (Q28), vault HTTP routes (Q29), `api-token` rotation and revocation (Q30), master-key rotation at startup (Q31).

## 2026-11-23 — Audit verify route and concurrent appends (F1-T18 partial, Q07, Q10)

- `GET /v1/audit/verify` (read scope): recomputes the chain through `Store.VerifyAudit` and returns `ok`, `entries_checked` and `first_break` (`seq`, `id`, `reason`; null when intact). Handler in `internal/api/audit.go`; `Config.Audit` is the verifier interface. OpenAPI: `verifyAudit` and `AuditVerification` (`first_break` nullable, OpenAPI 3.1 type list).
- Concurrent appends surfaced `SQLITE_BUSY` ("database is locked"). A deferred transaction reads the audit head and then writes; under WAL a stale snapshot fails at once and `busy_timeout` does not retry it. Fix: `_txlock=immediate` in the store DSN, so every transaction takes the write lock at `BEGIN`. Applies to all store transactions; reads outside transactions are unchanged.
- `internal/store/audit_concurrency_test.go`: 8 writers × 25 appends on a file database, no error, chain verifies with 200 entries. The same test failed before the fix.
- Not done: export (Q08), desktop view (Q09), decision on pre-0008 rows (Q11). The route exposes the first break only, never entry contents.

## 2026-11-23 — Vault audit events (F1-T10 partial, Q28)

- `internal/vault`: `Service.Set`, `Reveal` and `Delete` append an audit event before they act (`vault.set`, `vault.reveal`, `vault.delete`). Actor is `owner` with the owner id; subject `vault:<name>`; facts hold only the name. `NewService` now takes an `Auditor` and an id source; `store.Store` satisfies both (`AppendAudit`, `core.NewID`).
- Fail-closed: if the audit write fails, the action does not run and the error is returned. An attempt that fails afterwards (for example, deleting an unknown name) stays on record.
- Wired in `cmd/calango/main.go` and `internal/auth/token_test.go`.
- Tests: `internal/vault/service_audit_test.go` checks event order, that the plain value appears in no event, and that a refused audit blocks set, reveal and delete.
- Not done: vault HTTP routes (Q29), `api-token` rotation (Q30), master-key rotation at startup (Q31), audit export (Q08).
- Vault events carry no request context: in-process callers record as `owner`. A routine that reads `vault://` will need its own actor kind when that path exists.

## 2026-11-23 — Vault HTTP routes (F1-T10 partial, Q29)

- Routes in `internal/api/vault.go`: `GET /v1/vault` (names and dates), `PUT /v1/vault/{name}` (set; 200 with name and dates) and `DELETE /v1/vault/{name}` (204; 404 for an unknown name). All require full scope (`PermWrite`), as devices do. No route returns a value; there is no reveal over HTTP.
- Contract: tag `vault`, paths and schemas `VaultItem` and `VaultSetInput` (`value` is `writeOnly`) in `api/openapi.yaml`. Redocly lint valid.
- `api-token` is reserved: PUT and DELETE return 409 (`conflict`) until rotation (Q30) exists. Replacing or deleting it would change or lock out the token without a rotation.
- Error mapping: `vault.ErrInvalidName` and `ErrEmptyValue` now wrap `core.ErrInvalidInput` (400; they were 500). `Reveal` and `Delete` validate the name before writing the audit event, so an invalid name leaves no audit row.
- `internal/api/api_test.go`: `newTestEnv` returns the handler and the store; `newTestHandler` wraps it. `internal/api/vault_test.go` adds 7 tests.
- `cmd/calango/main.go` passes the vault service as `Vault`.
- Not done: `api-token` rotation (Q30), master-key rotation at startup (Q31), audit export (Q08).

## 2026-11-23 — Token rotation (F1-T14, Q30)

- `auth.LocalToken` (`internal/auth/token.go`) holds the live local token in memory. `Matches` compares in constant time; `Rotate` generates a token, stores it in the vault as `api-token` (the vault audits it as `vault.set`), and only then swaps it in. A mutex serialises rotations so the vault and memory always end on the same token. The vault write fails closed: if it fails, the old token stays live.
- Route `POST /v1/auth/rotate-token` (`internal/api/auth.go`): full scope (`PermWrite`); 200 with `{token}`, the only response that carries a token. Read and approve_only devices get 403. Contract in `api/openapi.yaml`.
- `api.Config.Token` is now `*auth.LocalToken` instead of a string. `cmd/calango/main.go` wraps the token it loads from the vault.
- The reserved-name guard for `api-token` still refuses PUT and DELETE. Rotation calls the vault service directly, so it is the only route that changes that item.
- Risk: if the rotation response is lost, the client has no token and no route returns the new one. Recovering needs a local operator; V1 has no tool for that. Accepted by the owner on 2026-11-23 and documented here and in `DEV/HANDOFF.md`.
- Not built: a route that revokes the local token without replacing it (it would leave no local client); master-key rotation at startup (Q31).

## 2026-11-23 — Usage known/estimated split (F1-T17, Q18)

- `GET /v1/usage` rows carry `cost_known_micros` and `cost_estimated_micros` next to `cost_micros` (still known plus estimated). The store sums each status with `CASE`; the schema already forces unknown cost to NULL.
- Changed: `usage.Row`, `store.UsageTotals`, `api.usageRow`, `api/openapi.yaml` (`UsageRow` and `GET /v1/usage` description). No migration.
- `TestUsageTotalsScopesToOwnerAndCountsUnknownCost` now includes an estimated row and checks the split.
- Not done: desktop panel (F1-T17) and projects group (no table in V1).

## 2026-11-23 — Audit export (F1-T18, Q08) and master-key rotation at startup (F1-T10, Q31)

- Q08: `GET /v1/audit/export` (read scope) returns JSONL (default; checkpoint line first, omitted for an empty chain) or CSV (entries only). Other formats return 400. The export is buffered and sent with `Cache-Control: no-store`. The contract is in `api/openapi.yaml`; reads are not audited.
- Q31: `CALANGO_VAULT_KEY_VERSION` (default 1), `CALANGO_VAULT_KEY_PREVIOUS` and `CALANGO_VAULT_KEY_PREVIOUS_FILE` (mutually exclusive) in `internal/config`; `vault.KeySources` drives `LoadMasterKeys`, which loads the previous key as version − 1. `cmd/calango` runs `Service.Rotate` before serving; each re-wrap is audited as `vault.rewrap` (name only) and fails closed. Startup refuses a row whose version has no configured key.
- Docs: ADR 0012 amendment (rotation procedure); `modules/config.md`; `modules/vault.md`; ROADMAP F1-T10; TODO_V1 Q01, Q04, Q05, Q08, Q11, Q31, F1-T10, F1-T14; HANDOFF; CONTEXT.
- Decisions recorded: Q11 accepted (pre-0008 rows verify as breaks; no reset). A lost response from `POST /v1/auth/rotate-token` leaves no token and no recovery route; accepted by the owner and documented. Q19 (`RemoteAddr`) deferred until VPS deploy.
- Not done: keyring-key rotation; owner acknowledgement of startup strictness (the daemon refuses to start without a master key); Q08 UI (Q09); commit and push of this batch (needs an explicit request).

## 2026-11-23 — Codex CLI harness (F1-T05 partial, ADR 0013)

- `internal/harness.Codex` is a `gateway.Provider` named `codex`. `Complete` runs `codex exec --ephemeral --ignore-user-config --ignore-rules --skip-git-repo-check --sandbox workspace-write -C <dir> -o <file> -m <model> -` with the prompt on stdin. Stdout and stderr are discarded; errors never echo CLI output. The final message is read from the `-o` file and streamed as one chunk. Usage is always estimated (about 4 characters per token) because the `--json` schema is unverified.
- `Available` checks the binary with `exec.LookPath` and runs `codex login status`. Signed out returns `ErrSignedOut`; `conversations` maps it to `failed/harness_signed_out` (`core.ErrCodeHarnessSignedOut`).
- Cancellation kills the CLI; `cmd.WaitDelay` (2 s) stops a child process from blocking `Wait`. Found by a test that took 30 s before the fix.
- Owner decisions used: isolation shares `~/.codex` (`compartilhar`); no real Codex model call (`nao`); sandbox `workspace-write` after the risk was shown.
- Not done: real run; `--json` parsing; daemon wiring (config `CALANGO_CODEX_BIN`, workdir); Claude Code CLI, OpenCode, Antigravity `agy`, direct API; killing the whole process group.

## 2026-11-23 — Maestro npm probe and D6 amendment (F1-T16 prep, Q24)

- ADR 0009 amendment: automatic update on by default (D6). Aligned: `ORQUESTRADOR_MAESTRO.md`, `PLANO.md` §14, `TECHNICAL_DELIVERY.md`, `SPECS/modules/maestro.md`, `TODO_V1.md` (D6 done, F3-T08 wording).
- Probed `@iapro/orquestrador-maestro-cli@1.1.1` outside the repo. Findings in `SPECS/modules/maestro.md` ("Probe findings"): `--home-path` confines files; telemetry id goes to `XDG_CONFIG_HOME`; `verify` requires the Codex target; JSON only from `status`, `context`, `route`, `usage`.
- Decided by owner (ADR 0014): isolated install in `<calango-home>/maestro/`, telemetry off, Codex target under its own `CODEX_HOME`. Calango never calls `update` (global npm self-update; see `SPECS/modules/maestro.md`).
- Applied `update --only codex --non-interactive` and `rollback --to` in a temp home with `ORQUESTRADOR_MAESTRO_SKIP_CLI_UPDATE=1`; `verify` passed after each. `version --check` is broken with this npm (array output). Global Maestro installs exist on this machine (`/usr/local`, root-owned; `~/.npm-global`), both 1.1.1; the probe did not change them.

## 2026-11-23 — Proxy-aware pairing limit (Q19, F1-T15)

- `/v1/pair` counts attempts per origin. The origin is the TCP peer, unless the peer is in `CALANGO_TRUSTED_PROXIES` (new in `internal/config`: comma-separated IPs or CIDRs, empty by default). Then it is the right-most `X-Forwarded-For` entry that is not a trusted proxy. A malformed chain falls back to the peer, so a spoofed header cannot leave its bucket.
- Tests: `internal/api/pairing_origin_test.go` (trusted proxy keys by client; untrusted peer ignores the header; malformed chain falls back) and `internal/config/trusted_proxies_test.go` (parse and reject). `newTestEnvWith` in `api_test.go` takes the trusted list; `newTestEnv` is unchanged for callers.
- Not done: no real reverse proxy run (no VPS yet); the setting is not checked against the proxy's actual address.

## 2026-11-23 — Conversations and runs routes (Q22, F1-T07/T13) and daemon wiring

- Routes in `internal/api/conversations.go`: `GET/POST /v1/conversations`, `GET/POST /v1/conversations/{id}/messages` (POST needs `Idempotency-Key`, 201 `{message, run}`), `GET /v1/runs/{id}`, `POST /v1/runs/{id}/stop` (204; 409 on a finished run), `POST /v1/runs/{id}/retry` (`Idempotency-Key`, body `instructions`, 201). `Config.Conversations` is required by `NewHandler`.
- Daemon: `cmd/calango` builds `gateway.New` with the store as ledger and `harness.NewCodex("codex", <data-dir>/workspace)` (directory created at startup); a `conversations.Service` with provider `codex` runs `Start(ctx)` before serving. No new config variable. The price table is empty, so cost stays unknown.
- Contract: `api/openapi.yaml` conversations and runs paths match the code. Added `Run`, `RunDetail`, `Step`. Removed `thread_id`, `unread`, `before` (not implemented) and the `steer` path (not routed).
- Tests: `internal/api/conversations_test.go` (create, list, send, idempotency; stop and retry). `api_test.go` uses a `blockingGateway` so runs stay queued or running without a provider.
- Not done: steer/guide (owner decision); one reply at a time per agent; a real Codex run; WebSocket; routines routes; approvals card; direct Anthropic provider; Maestro.
- Note: line-range `PUT` on `conversations_test.go` corrupted the file twice; it was rewritten whole.

## 2026-11-23 — WebSocket `/v1/events` (Q23, F1-T13)

- Dependency: `github.com/gorilla/websocket` v1.5.3 (BSD-2-Clause, `go 1.12` directive). `coder/websocket` rejected: its `go.mod` needs Go 1.23 and the project declares `go 1.22`. Decision in ADR 0015; `NOTICE` updated.
- `internal/events/hub.go`: in-process hub. `Publish` never blocks; subscriptions filter by owner; buffer 64; a full buffer closes that subscription.
- `internal/conversations/service.go` publishes `message.created` inside admission, before dispatch, so it precedes `run.started`; `run.started`; `run.step` after a successful `AppendStep`; `run.finished` from `execute` and from the queued-stop path.
- `internal/api/events.go`: `GET /v1/events`. Bearer header through `need(PermRead)`; subscribes before the upgrade; ping every 25 s, pong wait 60 s, write wait 10 s; close 1013 on overflow.
- Wiring: `api.Config.Events`; `cmd/calango` creates one hub shared by the conversations service and the API.
- Contract: `api/openapi.yaml` `/v1/events` (101, 401, 403; frame described in the description).
- Not done: Electron relay (main process to renderer over IPC); reconnect loop in desktop and mobile clients; replay; routine events.

## 2026-11-23 — Direct Anthropic provider (F1-T04)

- `internal/anthropic/anthropic.go`: `gateway.Provider` named `anthropic` over `POST /v1/messages` with `stream: true`. Key read from the vault item `anthropic-api-key` for each call, sent only in `x-api-key`. Parses SSE `data:` lines: `text_delta` chunks, exact input and output tokens (`message_start`, `message_delta`), `error` events, and a stream that ends without `message_stop` (error).
- Errors: non-200 gives status and API error type and message; the key and headers are never included.
- Daemon: `cmd/calango/main.go` registers the provider in the gateway next to Codex. Conversations still use `Config.Provider = codex`, so no agent reaches Anthropic yet. Choosing the provider per agent or per node is an open decision.
- No new dependency. No ADR: the provider follows ADR 0013.

## 2026-11-23 — Provider chain per agent (Q32, ADR 0016)

- Migration `0010_provider_chain.sql`: adds `provider_accounts` and `agent_providers`; copies `agents.harness`/`agents.model` into a `default` account and chain position 0, then drops both columns. Agents with an empty model get no chain and must be saved again with `providers`.
- `internal/core`: `ProviderAccount`, `ProviderRef`; `Agent.Providers` (1–8 entries). `StepProviderAttempt`, `ErrCodeNoProvider`.
- `internal/conversations`: `call` walks the chain. Fallback only before the first output chunk and only for `ErrUnavailable`, `harness.ErrNotInstalled` and `harness.ErrSignedOut`. Each attempt is one gateway call and one `provider_attempt` step.
- `internal/gateway`: `ErrUnavailable` sentinel; `Config.Resolve` resolves keys `<kind>/<label>` that are not in the static map. Static registrations removed from `cmd/calango`.
- `internal/providers`: account CRUD and resolution. Codex accounts get `<data-dir>/accounts/codex/<label>`; `default` shares `~/.codex`. Anthropic accounts use `anthropic-<label>-api-key`.
- `internal/harness`: Codex sets `CODEX_HOME` only when an account home is given.
- `internal/anthropic`: `NewAccount` and `KeyName` per account; HTTP 429, 5xx, transport and stream failures wrap `ErrUnavailable`.
- `internal/api`: agents take `providers` (replaces `harness`/`model`); `GET/POST /v1/provider-accounts`, `DELETE /v1/provider-accounts/{id}`; `api/openapi.yaml` updated.
- Not done: steer, Q33 routines, real CLI or API runs.

## 2026-11-23 — Steer (`POST /v1/runs/{id}/steer`, Q22, F1-T07)

- Migration `0011_steer.sql`: `runs.steer_of` references the run being steered (`ON DELETE SET NULL`). `internal/core`: `Run.SteerOf`; `internal/store`: column in `runColumns`, insert and `scanRun`.
- `internal/conversations`: `Service.Steer`. Queued run: stopped at once and a continuation is admitted. Running run: `activeRun.steer` holds the continuation (pre-generated ID, not persisted) and the run is cancelled; `execute` admits the continuation after `FinishRun`. `Stop` on a run with a pending steer and steer on a run that is not queued or running return `ErrConflict`. Continuation prompt: `steeredPrompt` (reply so far, guidance).
- `internal/api`: `POST /v1/runs/{id}/steer` (write scope), body `{"guidance": "..."}`, `Idempotency-Key`, 202 with the continuation run; `steer_of` in run JSON; `api/openapi.yaml` path and `Run.steer_of`.
- Tests: `internal/conversations/steer_test.go` (queued steer, running steer with partial output and repeated key, rejections); `internal/api/conversations_test.go` (400 without key, 202, same ID on repeat, 404).
- Not covered: Stop on a running run with a pending steer (code path only); crash while a steer is pending (loss documented, not tested); a real model call (not authorised).

## 2026-11-23 — Routines fire agent runs (Q33, F1-T09/T13, ADR 0017)

- Migration `0012_routine_conversation.sql`: `routines.conversation_id` (FK to `conversations`) and `ended_at`. Ended routines are excluded from `ListDueRoutines`.
- `core.Routine.Validate`: whitespace-only prompt rejected (ErrInvalidInput) before the DB CHECK would return 500. `core/routine_signal.go`: `RoutineMessage(prompt)` appends the control contract; `ParseRoutineSignal` reads the last valid `ROUTINE: {...}` line; missing or malformed means `had_news=true, finish=false`.
- `internal/routines`: `Service.Update` (PATCH, including `paused` and `clear_quiet`), `Delete`, `RunNow` (`@manual:<id>` occurrence for the next tick), `Occurrences`, `Get`. `ChatExecutor` (`executor.go`) opens one DM per routine and sends one agent message per attempt through `conversations.WaitRun` (polls every 250 ms), idempotency key `<OccurrenceID>#<Attempt>`. Ended routines reject pause, run now and update with 409; finish on a failed attempt is ignored. `runMu` serializes ticks; `mu` guards store state; executions run without `mu`.
- `internal/api/routines.go`: `GET/POST /v1/routines`, `GET/PATCH/DELETE /v1/routines/{id}`, `POST /v1/routines/{id}/run` (202), `GET /v1/routines/{id}/runs`.
- `cmd/calango/main.go`: routine service wired with the agent service; `routineTick = 30s`; `routine.fired` and `routine.failed` published to the events hub (`hub.Publish` with `ev.Kind`).
- `api/openapi.yaml`: routine paths and schemas (`RoutineInput`, `RoutinePatch`, `Routine`, `RoutineOccurrence`). Nullable fields use OpenAPI 3.1 `type: [..., "null"]`. The trigger enum is `cron, interval` (the `event` value was removed; events are phase 3).
- Known limits: a crash between `CreateDM` and saving `conversation_id` creates a second DM; a failed attempt's run stays in the conversation; `waiting_approval` counts as a failed attempt until Q15; one occurrence at a time per node; V1 wiring runs on the PC node only.

## 2026-11-23 — Maestro validation (Q24, validation only)

- Ran Calango's settings on npm `@iapro/orquestrador-maestro-cli@1.1.1` in the throwaway `/tmp/maestro-calango-probe` (not deleted; owner approval needed). Results in `SPECS/modules/maestro.md`.
- Corrected probe note: `verify` takes `--home-path` only. `--only codex` also writes a Codex desktop plugin under `<home>/.codex/plugins/`.
- Still open: the Calango code for F1-T16 (`internal/maestro`, routes, audit, auto-update) and the `CODEX_HOME` check (ADR 0014).

## 2026-11-23 — Approvals check run ownership (Q15 slice, F1-T08)

- `approvals.Request` now loads the run (`Repository.GetRun`, store method already existed) and rejects a run that belongs to another agent with `core.ErrInvalidInput`. Previously any run ID of the owner was accepted.
- Test: `TestRequestRejectsRunOfAnotherAgent` in `internal/approvals/service_test.go`.
- Still open in Q15: `waiting_approval` run state, reconciliation of approved actions without receipt, and any caller of `Request`/`Execute` (depends on MCP, Q20; Go minimum decision pending).

## 2026-11-23 — Go minimum 1.23 (MCP prerequisite)

- Owner chose option A: Go minimum 1.23, so `github.com/modelcontextprotocol/go-sdk` v1.3.0 can be used (v1.4.0 needs Go 1.24; v1.5.0 and later need 1.25).
- `go.mod` declares `go 1.23.0`. CI, `deploy/Dockerfile.node`, CONTRIBUTING, the three READMEs and the DEV docs moved to 1.23. go-sdk stays `// indirect` until code imports it.
- Still open: MCP (Q20) and the NOTICE entries for go-sdk and its dependencies, which are added once code imports them.

## 2026-11-23 — MCP gateway per turn (Q20, F1-T12, ADR 0018)

- Migration `0013_mcp.sql`: `connectors` (per owner, `stdio` or `http`, `UNIQUE (owner_id, name)`) and `agent_tools` (agent × connector, access `read` or `write`; no row means none).
- `internal/tools`: register, list, get, delete, grants (`Grant`, `Revoke`, `Grants`), `Catalog`, `RemoteTools`, `CallRemote`. Stdio children get only `PATH` and `HOME`. Unreachable connectors return `ErrUnavailable`.
- `internal/mcpgw`: `/mcp` on the root mux, outside the API bearer middleware; per-turn 32-byte token (SHA-256 kept in memory); `read` exposes `readOnlyHint` tools only; `write` calls go through `approvals.Request` with `core.ActionConnectorWrite` and wait up to 2 min; `Execute` runs the stored snapshot and keeps the result for 1 h.
- Token goes in env `CALANGO_MCP_TOKEN` (Codex `bearer_token_env_var`), not in the URL. Deviation from the spec's secret URL, recorded in ADR 0018.
- `internal/harness/codex.go` adds the `mcp_servers.calango` config only when the run has grants. `internal/conversations` opens and releases one endpoint per run.
- API: `GET/POST /v1/connectors`, `DELETE /v1/connectors/{id}`, `GET /v1/connectors/{id}/tools`, `GET /v1/agents/{id}/tools`, `PUT /v1/agents/{id}/tools/{connectorID}`. OpenAPI updated (`connector_unavailable` 502).
- `cmd/calango/main.go` wires `tools.Service`, `mcpgw.Gateway`, the approvals executor for `ActionConnectorWrite`, and `/mcp`.
- NOTICE: go-sdk (MIT), jsonschema-go (MIT), go-humanize (MIT), uritemplate/v3, google/uuid, x/exp, bigfft, modernc libc/memory/mathutil (BSD-3-Clause).
- Not done: connector credentials; Calango's own tools (`memory_save` and the rest); a custom refusal for hidden tools; restart durability for results; Q15 `waiting_approval` (the run stays `running` while a write waits).

## 2026-11-23 — Waiting approval and reconciliation (Q15, F1-T08)

- `internal/mcpgw`: a write call that opens a pending approval moves its run `running` → `waiting_approval`, and back to `running` when the call stops waiting (decision, timeout or cancel). `moveRun` logs and continues if the run is not in the expected state.
- `internal/store`: `SetRunStatus` is a compare-and-set and rejects transitions outside the allowed ones (`waiting_approval` → `failed` tested). `CountRunningRuns` counts `waiting_approval` runs, so they keep their slot. `InterruptRunningRuns` ends a `waiting_approval` run left by a crash as `failed`. Test: `internal/store/run_status_test.go`.
- `internal/conversations.WaitRun` keeps waiting while a run is `waiting_approval`. Before, it returned at once, so a routine attempt that waited for approval counted as failed. Test: `TestWaitRunHoldsWhileRunWaitsForApproval`.
- `internal/approvals.Reconcile`, called at startup in `cmd/calango/main.go` before the conversations service starts: closes approved actions without a receipt as `failed` with a reason, audits `approval.reconciled`, and never retries them. Test: `TestReconcileClosesApprovedWithoutReceiptAndNeverRetries`.
- `Execute` refuses an approval that already has an outcome with `ErrConflict`, so a reconciled action cannot run later.
- Test: `TestWriteHoldsRunWhileWaitingForDecision` (mcpgw) checks the run is `waiting_approval` while the write waits and `running` after the decision; the timeout test checks it is `running` again after the call stops waiting.
- The in-conversation card is implemented; a mocked renderer smoke covers card rendering/decision dispatch, but the real pending→decision API/action flow remains unverified.
- Not covered: stopping a run that is `waiting_approval`; a daemon restart while a write waits (only the store transition is tested); a real model call (not authorised).

## 2026-11-23 — memory_save through the gateway (Q17, F1-T11, ADR 0018 amendment)

- `internal/mcpgw`: built-in tool `memory_save` (`text` required; `kind` fact, preference or decision). Scope and agent come from the turn; the arguments cannot pick another agent. It calls `memory.Service.Create` with `ScopeAgent`. Empty text returns a tool error.
- `Open` returns an endpoint when the agent has grants or a memory service is wired. An agent without grants gets only `memory_save`. The daemon always wires memory (`cmd/calango/main.go`).
- `conversations.Tools` doc updated to the new nil rule.
- Tests: `TestOpenWithoutGrantsOffersOnlyMemorySave` (replaces "no grants, no endpoint"), `TestMemorySaveWritesAgentMemoryOnly`; `TestReadGrantAdvertisesAndRunsOnlyReadTools` now expects `memory_save` too.
- Not done: purge schedule; `project` scope. The owner addendum (channel and agent actions as memory base) is recorded in `DEV/SPECS/modules/memory.md` ("Adendo") with its design open; no code.

## 2026-11-23 — Desktop functionality and chat replies

- The owner's report that desktop actions did nothing was treated as a functional repro. The isolated Electron run showed `harness_signed_out`: npm/npx had prepended `node_modules/.bin`, whose `codex` shadowed the selected CLI. The daemon now removes those entries from the child `PATH`.
- Agent, conversation and routine forms select the first available provider/agent after loading. Pause/resume, approval decisions, routine toggles and run-stop failures now appear in the UI instead of disappearing.
- Successful message-triggered runs now persist their joined text steps as an agent-authored conversation message in the same transaction as run completion; `message.created` is published after commit. This makes replies visible in polled conversation history after reload.
- Desktop smoke used a disposable data directory and a fake Codex CLI: create agent, pause/resume, create a conversation, send a message and see the fake reply, create/pause a routine, and see its usage row. No real model call or production data was used.
- Isolated Chromium renderer smoke used mocked IPC/API with a simulated waiting run and pending approval: action/target rendered, and Approve, Always allow, and Decline each reached the mock decision endpoint with HTTP 200. This does not verify the daemon approval path or action execution.
- V1 remains incomplete: Windows packaging, live approval-flow validation, event reconnect/replay, settings/audit UI, verified provider pricing and real-model validation remain open.
- Rebuilt canonical `bin/calango` and restarted Electron with the same isolated profile/data directory already in use; `/healthz` on `127.0.0.1:7317` returned 200 and CDP stayed disabled. The default `~/.config/calango` database was not opened or modified.

## 2026-11-23 — V1 TODO continuation

- Replaced `OpenMemory`'s shared URI with a unique in-memory SQLite URI per store. Added `TestOpenMemoryCreatesIsolatedDatabases`; separate stores no longer see each other's rows.
- Fixed the desktop translation build: the i18n generator now emits a small, typed tray catalog under `apps/desktop/src/main`, inside the main TypeScript root. Removed a stale CommonJS `messages.js` that shadowed the renderer's generated TypeScript import.
- Implemented Linux startup-at-login through an XDG autostart `.desktop` entry; Electron's generic login-item setting did not apply on Linux. Added a Node regression test and `npm run test:main -w apps/desktop`.
- Continued F1-T16 integration: manager/API/scheduler, V1 install/verify, agent link, update, audit and rollback are present; kept the task partial because per-version `CODEX_HOME` conflicts with ADR 0014's stable-home decision.
- Updated screen and conversation behavior/status: audit/settings UI, one-active-run-per-agent FIFO, memory purge scheduler, approval API integration, secret-canary coverage and bounded API requests. Phase 1 exit remains unmet.
- Reviewed D5 against official sources and recorded BSD-3-Clause, Personal plan limits and the unresolved commercial-embedding Terms question in ADR 0005. No Tailscale dependency added.
- Reviewed the site at 1440 px desktop and 390 px mobile in light/dark. Mobile has no horizontal overflow; plan cards present phone/cloud/team features that remain roadmap items and need owner copy review.
- Verification: Go race suite and vet; desktop typecheck/build; i18n drift check; main-process startup tests; isolated Electron settings/audit/export smoke. No real model call or production user data. No commit/push.

## 2026-11-23 — TODO status reconciliation

- Classified owner/platform/engineering/phase-gated items; marked REST reconnect and API routes complete without WebSocket replay per ADR 0015; confirmed Calango telemetry is absent and Maestro telemetry is forcibly disabled. Corrected stale CI, Redocly, SDK/i18n and desktop status text.
- Added `DEV/BACKUP_RESTORE.md` and a migration restore test for an offline pre-migration SQLite snapshot. Previous-binary compatibility and OS-keyring recovery stay open.

## 2026-11-23 — Desktop launch, checks and push

- `gofmt -w internal/api/server.go`: struct alignment only. `.gitignore` now ignores `/bin/` (local daemon binary) and `apps/desktop/dist/` (build output). `package-lock.json` and `packages/i18n/generated/arb/*.arb` stay tracked.
- Commit in scope-based commits and push `main` on the owner's explicit request ("sobe no github branch principal"). Diff checked for env/key files before commit.
- Stopped at the phase boundary: Phase 1 exit not met; phases 2–6 gated; open items in `DEV/TODO_V1.md` "Status and resume point".
- Stopped a leftover smoke Electron instance (CDP 9225, isolated `/tmp` profile) before launch. Desktop relaunched on display `:1` with the shell environment (`CALANGO_VAULT_KEY_FILE`, `CALANGO_ADDR=127.0.0.1:7399`), without CDP. Data directory not overridden in this session and not checked.
- Verification and the unverified items are in `DEV/VERIFY.md`, entry "2026-11-23 — Desktop launch, checks and push".

## 2026-11-23 — F1-T16 stable Codex home (ADR 0014)

- `internal/maestro/manager.go`: the Codex home is now `<calango-home>/maestro/.codex` for every release. Each release is installed and verified in its staging home `versions/<v>/home`; only a verified release is activated into the stable home (`activate`). Install and rollback restore the currently selected release into the stable home when activation or audit fails (`restoreHome`). Removed `versionHome`; `env` takes the home argument.
- Why: a per-version home signed the person out on every auto-update and rollback, and ADR 0014 decision 3 requires a stable home. Copying `auth.json` stays rejected (ADR 0013).
- Migration: logins stored under earlier per-version homes are not moved or copied. The person signs in to Codex again once. Owner policy decision still open.
- Tests: `TestCodexHomeRequiresActiveVerifiedRelease` now expects the stable home; new `TestCodexHomeIsStableAcrossReleases` and `TestFailedReleaseVerificationNeverWritesActiveCodexHome`.
- Linked agents already receive the stable home: `cmd/calango/main.go` passes `maestroManager.CodexHome` to the conversation service.
- F1-T16 stays partial: no real Codex session has run against the stable home.

## 2026-11-25 — Small site favicon

- Worked on #49's icon scope: created a simplified SVG lizard silhouette in the brand's moss/post-it palette, linked it as the site favicon, and retained PNG fallback and Apple touch icon.
- Updated `DEV/BRAND/BRAND.md`, TODO, handoff, context, and verification records. This does not publish the site or complete #49; Q27's domain/legal/pricing/HTTPS decisions remain open.
- Verified a 32 px Chromium render and the built site serving the favicon with HTTP 200 (`image/svg+xml`); page title was present and browser errors were zero.
- No issue was closed or assigned. No commit or push was made.
