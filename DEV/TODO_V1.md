# TODO — Calango V1 ready and working

Generated from all the `DEV/` documentation on 2026-11-23. V1 scope: **phases 0 to 6** of the [ROADMAP](SPECS/ROADMAP.md), plus the integration of the [Orquestrador Maestro](ORQUESTRADOR_MAESTRO.md). Phase 7 (own relay, external channels, voice, marketplace) comes **after V1**.

Competitor research and positioning: [COMPETITIVE_LANDSCAPE.md](COMPETITIVE_LANDSCAPE.md). CI, release, migration and rollback flow: [TECHNICAL_DELIVERY.md](TECHNICAL_DELIVERY.md). The research does not change open decisions or turn planned capabilities into implemented ones.

Legend: `[ ]` pending · `[x]` done · `⛔` blocked by an owner decision · `→` source in the documentation.

Execution order: each block starts only when the previous one meets the exit criterion of the ROADMAP.

---


## Queue: everything still to do (ordered)

Blocked by owner decision; do not start before the answer:
- [ ] Q01 Owner confirms ADR 0011 status (recorded as "accepted"; legal review D1b open) or sets it to "proposed".
- [ ] Q02 Owner answers D3 (first harness). Unblocks F1-T04/05, F1-T07 guide/steer, F1-T09 executor, F1-T11 `memory_save`, `/v1/conversations*`, `/v1/runs/*`, `/v1/routines*`.
- [ ] Q03 Owner answers D2, D4, D5 and D6 (section 0).

Git (only on explicit owner request):
- [ ] Q04 Commit and push the uncommitted batch (ADR 0011, audit chain, docs, NOTICE, `third_party/agenticos/`). Nothing committed since `6c62ceb`.
- [ ] Q05 Run GitHub CI on the pushed commit; it has never run.
- [ ] Q06 Delete `/tmp/agenticos` after the P1–P3 ports (ask first; it is a temporary clone).

P1, audit (F1-T18):
- [x] Q07 HTTP route to verify the audit chain. Done: `GET /v1/audit/verify` (`internal/api/audit.go`, read scope); `verifyAudit` and `AuditVerification` in `api/openapi.yaml`; Redocly lint passes.
- [ ] Q08 Audit export; format in `DEV/SPECS/modules/approvals.md`.
- [ ] Q09 Audit view in the desktop (with F1-T23).
- [x] Q10 Test concurrent appends under load. Done: `_txlock=immediate` in the store DSN; `internal/store/audit_concurrency_test.go` (8 writers × 25 appends) passes. Before the change, `SQLITE_BUSY` reached callers.
- [ ] Q11 Decide the path for databases with audit rows from before migration 0008 (today: break reported from the first row). Accept for pre-release or reset.

P1, vault (F1-T10):
- [x] Q12 Check the licence of OS keychain library candidates (e.g. `github.com/zalando/go-keyring`) before adding; update `NOTICE`. Done: `go-keyring` v0.2.8 (MIT) and its dependencies are listed in `NOTICE`.
- [ ] Q13 Port `core/vault.py` (343 lines) to `internal/vault`: Windows Credential Manager, macOS Keychain, Linux Secret Service; refuses plain text; ADR 0011 header. Partial: envelope, service, store, bootstrap, audit (Q28) and HTTP (Q29) done (ADR 0012); the keyring backends compile for Windows and macOS but have not run there.
- [x] Q14 Move the API token into the vault. Done: stored as `api-token`, legacy file imported and removed; rotation replaces it (Q30).
- [x] Q28 Vault audit events for set, reveal and delete, in the audit chain (F1-T18); record names and actors only, never values. Done: `Service.Set`, `Reveal` and `Delete` append an event before acting (`vault.set`, `vault.reveal`, `vault.delete`; facts hold the name only); a refused audit write refuses the action (`internal/vault/service_audit_test.go`).
- [x] Q29 Vault HTTP routes: list names and dates, set, delete; scopes in `api/openapi.yaml`; no route returns a value; Redocly lint. Done: `GET /v1/vault`, `PUT /v1/vault/{name}` and `DELETE /v1/vault/{name}` (`internal/api/vault.go`; full scope only; 404 for unknown names). Responses carry name and dates, never the value. The reserved name `api-token` returns 409 until Q30. Tests: `internal/api/vault_test.go`; Redocly lint valid.
- [x] Q30 Rotation of the local `api-token`: a rotate action replaces the vault item and the live token; the old token fails on the next call. Done: `POST /v1/auth/rotate-token` (full scope; `api/openapi.yaml` operationId `rotateToken`), `auth.LocalToken` (`internal/auth/token.go`), `internal/api/auth.go`. The new token is returned once and is never stored in plain text. No separate revocation route: the rotate action revokes the old token, and revoking without a replacement would leave no local client. Tests: `internal/api/auth_test.go`, `internal/auth/token_test.go`; binary smoke (old 401, new 200, both after restart).
- [ ] Q31 Master-key rotation at startup: `LoadMasterKeys` supplies only version 1; wire `Service.Rotate` when a second key version is configured.

P2:
- [ ] Q15 Approvals: card, `waiting_approval` on runs, reconciliation of approved actions without receipt (F1-T08). Reference: `services/approvals.py` (883 lines) and `agent` (143).
- [ ] Q16 Routines: HTTP routes, client events, manual run, `routine_finish`, event triggers (F1-T09). Reference: `routine_actions.py` (505), `routine_delivery.py` (241), `routine_authority.py` (674), `routine_sources.py` (1106); port only what V1 uses.
- [ ] Q17 Memory: `memory_save` tool (needs D3), purge schedule, project scope decision (F1-T11). Reference: `repositories/memory.py` (291), `agents/memory_scope.py` (125).
- [ ] Q18 Usage: desktop panel (F1-T17). Known/estimated split done in `GET /v1/usage` and the store query (`cost_known_micros`, `cost_estimated_micros`). Reference: `services/usage_report.py` (365).
- [ ] Q19 Pairing: make `/v1/pair` rate limit proxy-aware (keys on `RemoteAddr` today) before any VPS deployment (F1-T15).

P3:
- [ ] Q20 MCP (F1-T12): check go-sdk licence (moving MIT to Apache-2.0) before adding; connect server, gateway per turn, scopes none/read/write. Reference: `agents/mcp.py` (1040), `services/mcp_connection.py` (3655), `services/mcp_audit.py` (517), `agents/mcp_oauth.py` (676); port in slices.
- [ ] Q21 Screens (F1-T23): routines widget reference `dashboard/widgets/routines.tsx` (225).

Blocked by D3 (conversations and events):
- [ ] Q22 Conversations and runs HTTP routes, guide/steer, one reply at a time per agent (F1-T07, F1-T13). Depends on Q02.
- [ ] Q23 WebSocket `/v1/events`: choose the dependency; check its licence and Go 1.22 support; no SSE substitute (F1-T13).

Maestro:
- [ ] Q24 Maestro (F1-T16): validate install destination, JSON format and non-interactive `update`; D6 default.

Verification and site:
- [ ] Q25 Docker bridge networking test on a compatible machine (only `--network host` verified).
- [ ] Q26 Site review on phone and in dark mode.
- [ ] Q27 Owner site items: CLA legal name, contacts (`licenca@`, `seguranca@`, `conduta@`), Caddy domain, Terms of Service link.

## 0. Owner decisions (block parts of V1)

- [x] D1 Calango code will be **open** (decided by the owner on 2026-11-23).
- [x] D1a License: **AGPL-3.0** + commercial license + CLA + trademark policy (ADR 0010). Files created.
- [ ] D1b Legal review of `CLA.md`, `COMMERCIAL.md` and `TRADEMARKS.md`; fill in the maintainer's legal name and the contacts (`licenca@`, `seguranca@`, `conduta@`).
- [ ] ⛔ D2 Remote screen control: Pro or only Team. → PLANO §14, PRODUCT_HYBRID
- [ ] ⛔ D3 First MVP harness: Claude Code, Codex or direct API. → `SPECS/modules/harness.md`
- [ ] ⛔ D4 Domain and Go module name (`calango` is a placeholder). → HANDOFF
- [ ] ⛔ D5 Check the `tsnet` license and Tailscale free-plan limits. → ADR 0005
- [ ] ⛔ D6 Maestro automatic update: confirm the default (suggested: off). → ADR 0009

---

## Phase 0 — Foundation

- [x] Go 1.27.4 installed in `~/.local/opt/go` (SHA-256 verified against go.dev).
- [x] `go vet ./...` clean.
- [x] `go test ./...` passes (4 tests in `internal/core`).
- [x] `gofmt -l` with no output.
- [x] `calango` starts and `GET /healthz` returns 200; POST returns 405.
- [x] F0-T04 CI on GitHub Actions: vet, test, build linux/windows/darwin on amd64 and arm64. Workflow written; six local builds passed. Confirm the GitHub run after the first authorized push.
- [x] F0-T05 `internal/config`: reads `CALANGO_*` in one place and validates at startup. `CALANGO_ADDR` keeps the loopback default and validates host/port before the scheduler.
- [x] F0-T06 Semantic validation of OpenAPI in CI with Redocly CLI 2.81.0 pinned; local lint passed with no problems or warnings.
- [x] F0-T07 Deterministic generator from `tokens.json` to CSS and TypeScript/Dart constants; CI fails if the generated files are missing or drift. The i18n catalog generator comes with F1-T26.
- [x] Token contract validation: `node scripts/generate-tokens.mjs --check` confirms the three generated artifacts. Flutter composes `ThemeData` from the Dart tokens in F2-T03.

---

## Phase 1 — Local desktop MVP (Windows first)

### Core
- [x] F1-T01 `internal/store`: SQLite (`modernc.org/sqlite`), WAL, immutable migration runner, in-memory tests. → DATA_MODEL, ADR 0002
- [x] F1-T02 Initial migrations: agents, conversations, messages, runs, run_steps, usage, audit_log.
- [x] F1-T03 `internal/gateway`: `Provider` interface, budget check before the call, usage after, fake provider. → `modules/gateway.md`, ADR 0007
- [ ] F1-T04 First provider: direct Anthropic API (HTTP with streaming); key comes from the vault.
- [ ] F1-T05 `internal/harness` with the adapter for the harness chosen in D3. → `modules/harness.md`
- [x] F1-T06 Agents: CRUD, global pause, monthly limit (service and store; HTTP wiring in F1-T13/T15, pause enforcement on runs in F1-T07). → `modules/agents.md`
- [ ] F1-T07 Conversations and runs: DM, persisted/streamed steps, stop, guide, retry with instructions; reconnection restores state and resending does not duplicate the run. → `modules/conversations.md`. Partial: guide and reconnection pending.
- [ ] F1-T08 Approvals: card with an immutable canonical payload, digest checked before execution, decision, separate receipt, "always approve", 24 h expiry. → `modules/approvals.md`; source ADR 0011 (P2). Partial: domain, store, service and tests done; card, HTTP and `waiting_approval` pending (F1-T13).
- [ ] F1-T09 Routines: cron and interval, ID per occurrence, idempotent retries with backoff, notify_on, quiet window, missed execution without repetition. → `modules/routines.md`; source ADR 0011 (P2). Partial: domain, store, service and tests done; HTTP, client events, real executor and `routine_finish` pending (F1-T13 / D3).
- [ ] F1-T10 Vault: Windows Credential Manager, macOS Keychain, Linux Secret Service; refuses plain text. → `modules/vault.md`; source ADR 0011 (P1, `core/vault.py`). Partial: envelope, store, service and master-key bootstrap done (ADR 0012); the keyring backend compiles for Windows, macOS and Linux but has not run on Windows or macOS; audit (Q28) and HTTP (Q29) done; master-key rotation (Q31) pending.
- [ ] F1-T11 Memory: save, list, edit, delete; injection under a token budget. → `modules/memory.md`; source ADR 0011 (P2). Partial: service, store and budgeted injection done; `memory_save` tool (D3), HTTP and project scope pending.
- [ ] F1-T12 MCP: connect a server, local MCP gateway per turn, scopes none/read/write. → `modules/tools-mcp.md`; source ADR 0011 (P3).
- [ ] F1-T13 API v1 per `api/openapi.yaml` and WebSocket `/v1/events`. Partial: REST for agents, memories, approvals, devices and usage (`GET /v1/usage`) served behind bearer auth and tested; conversations, runs, routines and `/v1/events` not implemented (D3).
- [ ] F1-T14 Local desktop token, stored in the vault. Partial: token is the vault item `api-token`, created on first start; a legacy `DataDir/api-token` file is imported and removed; every `/v1` route requires it. Rotation and revocation pending (Q30).
- [ ] F1-T15 Device pairing and authentication (local version, no phone yet). → `modules/pairing-auth.md`. Partial: pairing codes, device tokens, device list, revocation and scopes served and tested; QR code, mobile app and VPS administrator password pending.
- [ ] F1-T16 Maestro integration: install the V1 line, `verify`, link to agents, update card. → `modules/maestro.md`, ADR 0009
- [ ] F1-T17 Usage/cost panel per agent, routine, project, model and node; real, estimated or unknown cost is never shown as zero without evidence. → ADR 0007, DATA_MODEL `usage`; source ADR 0011 (P2, `services/usage_report.py`). Partial: `GET /v1/usage` with the five groupings, unknown-cost counting and the known/estimated split (`cost_known_micros`, `cost_estimated_micros`) served and tested; panel pending (desktop screens).
- [ ] F1-T18 Append-only audit of approvals, receipts, remote actions, permission changes and devices; no secrets or unnecessary content. → SECURITY. Partial: hash chain, checkpoint and `VerifyAudit` in the store (ADR 0011, `core/audit.py`); HTTP verify route, export and UI pending.

### Desktop (Electron)
- [ ] F1-T20 `apps/desktop`: Electron + Vite + React + TS strict; `contextIsolation`, `sandbox`, strict CSP. → `modules/desktop-electron.md`
- [ ] F1-T21 Main process starts and watches `calango`; restarts with backoff; state in the tray.
- [ ] F1-T22 `packages/sdk-ts` generated from OpenAPI; `packages/tokens` generating CSS.
- [ ] F1-T23 Screens: sidebar, conversations, approval card, routines, agents, usage, settings (you, agents, models, connections, vault), "About" with credit to Maestro. Source: ADR 0011 (P3, screen reference only).
- [ ] F1-T24 Tray, start with the system, global shortcut for quick conversation.
- [ ] F1-T25 **Signed** Windows NSIS installer + signed auto-update; Linux `.deb`; notarized macOS DMG.
- [ ] F1-T26 i18n: pt-BR, en and es from the first screen; single catalog in `packages/i18n`, with deterministic generation and drift checking in CI.
- [ ] Ship the `NOTICE` file inside the installer.

### Phase 1 exit criterion
- [ ] Install on Windows, talk to an agent, create a daily routine, approve an action, see the cost in the panel.

---

## Phase 2 — Phone

- [ ] F2-T01 Pairing by QR and single-use code (5 min), token stored in the keychain. → `modules/pairing-auth.md`
- [ ] F2-T02 Tailscale transport behind `internal/transport`; evaluate embedded `tsnet` (depends on D5). → ADR 0005
- [ ] F2-T03 `packages/sdk-dart` generated from OpenAPI; `ThemeData` composed from the Dart constants generated from `tokens.json`.
- [ ] F2-T04 `apps/mobile` Flutter: pair, nodes, "For you", conversations, approvals, routines, tasks. → `modules/mobile-flutter.md`
- [ ] F2-T05 Minimal Calango Cloud: accounts and devices, Postgres, migrations. → `modules/cloud.md`
- [ ] F2-T06 APNs/FCM push relay carrying only "there is news" and an opaque id; a test proves there is no text in the payload. → `modules/push.md`
- [ ] F2-T07 TestFlight (iOS) and internal testing (Android).
- [ ] The mobile app is available in the stores only after review of the remote-control guidelines.

### Phase 2 exit criterion
- [ ] Approve from the phone, away from home, an action requested by an agent on the PC, within 3 taps from the push.

---

## Phase 3 — VPS and first revenue

- [ ] F3-T01 Multi-architecture Docker image + one-line installer; compose and Caddy updated. → `DEPLOYMENT.md`
- [ ] F3-T02 Event log and replication between nodes (`events`, `peers`, commands to the owner). → `modules/nodes-sync.md`, ADR 0008
- [ ] F3-T03 Hybrid execution: `run_on` pc, vps and auto; owner handover when the PC goes down; return to the PC at the end of the run.
- [ ] F3-T04 Ed25519 license validated offline, 7-day tolerance, downgrade to Free without stopping or deleting anything. → `modules/licensing-billing.md`
- [ ] F3-T05 Billing: Stripe (web and desktop), store purchases (RevenueCat or direct), single plan source in the Cloud.
- [ ] F3-T06 `Entitlements` in the core: VPS, routines, projects, screen control, push.
- [ ] F3-T07 Interface shows an offline node: which one, since when, what is waiting for it.
- [ ] F3-T08 Automatic Maestro update (optional, off by default) with rollback and `verify`. → `modules/maestro.md`
- [ ] Terms of use and privacy policy published before billing starts.

### Phase 3 exit criterion
- [ ] With the PC off, an `auto` routine runs on the VPS at the scheduled time; the Pro subscription is charged and the VPS node is enabled.

---

## Phase 4 — ADE

- [ ] F4-T01 Projects: repository, node, agents, connectors, project `DEV/`. → `modules/ade.md`
- [ ] F4-T02 Worktree per task and its own branch; delivery includes the diff, the commands run, test status/output and artifacts; cleanup offers keep, PR or discard.
- [ ] F4-T03 Terminal: PTY in `calango` (`creack/pty`; ConPTY on Windows), WebSocket `/v1/pty/{id}`, xterm.js.
- [ ] F4-T04 Monaco editor, file tree, diff per worktree.
- [ ] F4-T05 GitHub: create branch, open PR, read CI status and attach reviewable run evidence. GitLab comes after V1.
- [ ] F4-T06 Preview server detected and opened in the internal browser (depends on F5-T01).
- [ ] F4-T07 Import skills from `.claude/` and `.codex/`.
- [ ] Safe list of shell commands; anything outside it requires approval.
- [ ] A project on the VPS editable from the desktop with the same editor and terminal.

### Phase 4 exit criterion
- [ ] Two tasks in the same project run in parallel in separate worktrees and produce two PRs.

---

## Phase 5 — Browser and computer use

- [ ] F5-T01 Tabs with `WebContentsView`; persistent profiles per agent. → `modules/browser.md`
- [ ] F5-T02 The agent drives the tab through CDP; take over and hand back without reloading.
- [ ] F5-T03 Barrier detection (robot check, login, cookies, empty page) with a warning line to the agent, without quoting the page text.
- [ ] F5-T04 Headless browser on the VPS (chromedp) with the same interface; separate image. → DEPLOYMENT, pending item
- [ ] F5-T05 Computer use (desktop apps), off by default, with a blocked-apps list. → `modules/computer-use.md`
- [ ] F5-T06 Screen control from the phone (paid, depends on D2), with a notice on the PC, audit, and ending in under 1 s.
- [ ] Guided Accessibility and Screen Recording permissions on macOS.

### Phase 5 exit criterion
- [ ] The agent fills in a form in the internal browser; the person takes over mid-way and hands it back.

---

## Phase 6 — Channels and teams

- [ ] F6-T01 Channels, threads and mentions. → `modules/channels-teams.md`
- [ ] F6-T02 Router: a cheap model picks 0 to 2 agents; it never grants permissions; the choice appears at the top of the answer.
- [ ] F6-T03 Members, roles (owner, admin, member, guest), per-channel permissions, delegated approval.
- [ ] F6-T04 Team plan in billing; agent shared between people.

### Phase 6 exit criterion
- [ ] Two members and three agents talk in a channel; a message without a mention triggers at most two runs.

---

## Cross-cutting (apply to all phases)

- [ ] Secret audit: a canary test guarantees that no secret appears in `run_steps`, logs or the API. → SECURITY
- [ ] API limits: bounded request bodies, rate limit on `/v1/pair`.
- [ ] Database backup and restore documented and tested, including restore before migration and application rollback compatibility.
- [ ] Export and deletion of personal data exercised; first run detects node state, model, harness, vault and connector without leaking content.
- [ ] Telemetry only with opt-in, without content.
- [ ] User documentation: first use, pairing, VPS, Maestro, privacy.
- [ ] Accessibility review (keyboard, screen reader, contrast) on desktop, mobile and the site.
- [x] Site: "Contribute to V1" section and footer with AGPL-3.0 (done 2026-11-23).
- [ ] Site: replace "coming soon" with real prices once defined; host it; simplified icon for small sizes (`DEV/BRAND/BRAND.md`).
- [ ] Publish the repository at `IAPro-Community/calango` (the local repository exists, without a commit; push only on explicit request).
- [ ] Validate `docker build` and `docker run` of the site `Dockerfile` and `deploy/Dockerfile.node` (blocked: Docker daemon unavailable in this environment; the user needs to grant daemon access).
- [ ] Validate the `.github/workflows/ci.yml` workflow after the first push.
- [ ] Full visual review of the site in light and dark with the browser pane visible (pending from the previous session).
- [ ] Remove any trace of reference code: review PRs against copies of agentes.chat (ADR 0006).

---

## Post-V1 (outside this TODO)

- Own relay (F7-T01).
- External channels: Telegram, WhatsApp, Slack, email (F7-T02).
- Voice (F7-T03), files and documents (F7-T04), marketplace (F7-T05).
- GitLab in the ADE.

---

## How to use this TODO

1. Confirm the decisions in section 0 with the owner before starting the block that depends on them.
2. Follow the phase order. Do not skip the exit criterion.
3. Mark `[x]` here and in `SPECS/ROADMAP.md` when each task is done, with evidence in `VERIFY.md`.
4. Record what was done in `WORKLOG.md` and the next step in `HANDOFF.md`.
