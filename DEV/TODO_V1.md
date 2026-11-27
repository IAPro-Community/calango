# TODO — Calango V1 ready and working

Generated from all the `DEV/` documentation on 2026-11-23. V1 scope: **phases 0 to 6** of the [ROADMAP](SPECS/ROADMAP.md), plus the integration of the [Orquestrador Maestro](ORQUESTRADOR_MAESTRO.md). Phase 7 (own relay, external channels, voice, marketplace) comes **after V1**.

Competitor research and positioning: [COMPETITIVE_LANDSCAPE.md](COMPETITIVE_LANDSCAPE.md). CI, release, migration and rollback flow: [TECHNICAL_DELIVERY.md](TECHNICAL_DELIVERY.md). The research does not change open decisions or turn planned capabilities into implemented ones.

Legend: `[ ]` pending · `[x]` done · `⛔` blocked by an owner decision · `→` source in the documentation.

Execution order: each block starts only when the previous one meets the exit criterion of the ROADMAP.

---


## Status and resume point (2026-11-23)

Stopped here: phases 2–6 not started; Phase 1 exit not met; V1 not complete.

Done in this round (evidence in `DEV/VERIFY.md`, entry "2026-11-23 — Desktop launch, checks and push"):
- ADR 0019 (desktop shell and token handoff) with the CORS addendum: `app://calango` only.
- `gofmt` fix in `internal/api/server.go`; `.gitignore` ignores `/bin/` and `apps/desktop/dist/`.
- Checks passed: `gofmt -l cmd internal` empty, `go vet ./...`, `go test ./... -count=1` (21 packages with tests), desktop `typecheck` and `build`, `node --test apps/desktop/test/startup.test.cjs` (2/2).
- Daemon kill recovery: SIGKILL on the daemon; the supervisor started a new process and `/healthz` returned 200.
- Desktop started on display `:1` with `npm run desktop:start`. Daemon `127.0.0.1:7399` answers 200; Electron process alive. Window visibility is not confirmed: no screenshot or window-list tool on this machine.
- Publication: `main` pushed to `origin` on the owner's request; the pushed head is the one shown by `git log origin/main -1`.

Remaining, by type:
- Owner-blocked: D1b legal review (CLA, COMMERCIAL, TRADEMARKS, legal name, contacts); Q03/D5 written Tailscale clearance or another transport; Q06 deletion of `/tmp/agenticos`; site prices, domain and Terms; memory capture/retention design; authorization for real Codex/provider runs and external approval actions; F1-T16 migration policy for logins stored by earlier per-version homes (current behavior: sign in again once).
- Publishing rule (owner, 2026-11-23): commits and pushes to `main` do not need a real Codex session or Codex login. This repo is public: commits must hold no personal data, credentials, local paths or private files. Check the diff before each push.
- Platform-blocked: Windows and macOS keyring/login runtime; Windows NSIS installer and signing (F1-T25); Linux `.deb`; macOS notarization; OS-level global shortcut (F1-T24); Docker bridge test (Docker daemon access denied).
- Engineering, not started: Codex JSONL event schema (F1-T05); F1-T16 real Codex check of the stable home (not authorized yet); other V1 harnesses (Claude Code CLI, OpenCode, Antigravity); MCP connector credentials, built-ins and result durability; previous-binary and OS-keyring backup restore; personal-data export/deletion; user docs; accessibility review; reference-code review; site icon.
- Partial phase 1 tasks: F1-T08, F1-T11, F1-T15, F1-T16, F1-T17, F1-T24, F1-T25.

Resume:
1. In the repo: `git status` and `git log -1` to confirm the head.
2. Checks: `go test ./... -count=1`, `npm run typecheck -w apps/desktop`, `npm run build -w apps/desktop`, `npm run test:main -w apps/desktop`.
3. Launch: `DISPLAY=:1 npm run desktop:start`. It inherits the shell's `CALANGO_VAULT_KEY_FILE` and `CALANGO_ADDR`; check `curl 127.0.0.1:$PORT/healthz` with the port from `CALANGO_ADDR` (7317 when unset). Do not enable CDP on the owner's session.
4. First engineering item: `DEV/HANDOFF.md`, "Next technical step", item 2 (F1-T16: the stable home is in code; the Codex check and migration policy remain).

Do not report V1 complete while the Phase 1 exit criteria are open.

## Queue: everything still to do (ordered)

Blocked by owner decision; do not start before the answer:
- [x] Q01 Owner accepted ADR 0011 (status "accepted"). Legal review (D1b) is still open.
- [x] Q02 Owner answers D3 (first harness): **Codex CLI first** (2026-11-23). Then Claude Code CLI, OpenCode, Antigravity (Google, CLI) and direct API in V1. Unblocks F1-T04/05, F1-T07 guide/steer, F1-T09 executor, F1-T11 `memory_save`, `/v1/conversations*`, `/v1/runs/*`, `/v1/routines*`.
- [ ] Q03 Owner decision after research: `tsnet` is BSD-3-Clause, but Tailscale's terms do not clearly authorize embedding/reselling Tailscale services in a commercial product. Personal plan is $0, unlimited user devices, up to 6 users, 3 ACL groups, 50 tagged resources and 1,000 ephemeral-resource minutes/month. Get written vendor/legal clearance or choose another transport; see ADR 0005.

Git (only on explicit owner request):
- [x] Q04 Commit and push the batch. Done: `6c62ceb..dee8efa` pushed to `origin` on the owner's request.
- [x] Q05 Run GitHub CI on the pushed commit. Done: run 45188079827 on `dee8efa` succeeded.
- [ ] Q06 Delete `/tmp/agenticos` after the P1–P3 ports (ask first; it is a temporary clone).

P1, audit (F1-T18):
- [x] Q07 HTTP route to verify the audit chain. Done: `GET /v1/audit/verify` (`internal/api/audit.go`, read scope); `verifyAudit` and `AuditVerification` in `api/openapi.yaml`; Redocly lint passes.
- [x] Q08 Audit export: `GET /v1/audit/export` (read scope), JSONL (default, checkpoint first) or CSV; buffered and `Cache-Control: no-store`. Done and tested; desktop audit screen Q09 is now complete.
- [x] Q09 Desktop audit screen: verify-chain status plus JSONL/CSV downloads. Isolated Electron smoke verified 23 entries and downloaded both formats; see `DEV/VERIFY.md`.
- [x] Q10 Test concurrent appends under load. Done: `_txlock=immediate` in the store DSN; `internal/store/audit_concurrency_test.go` (8 writers × 25 appends) passes. Before the change, `SQLITE_BUSY` reached callers.
- [x] Q11 Decided: pre-0008 audit rows stay unhashed and verify as breaks from the first row. Accepted by the owner for pre-release; no reset.

P1, vault (F1-T10):
- [x] Q12 Check the licence of OS keychain library candidates (e.g. `github.com/zalando/go-keyring`) before adding; update `NOTICE`. Done: `go-keyring` v0.2.8 (MIT) and its dependencies are listed in `NOTICE`.
- [ ] Q13 Port `core/vault.py` (343 lines) to `internal/vault`: Windows Credential Manager, macOS Keychain, Linux Secret Service; refuses plain text; ADR 0011 header. Partial: envelope, service, store, bootstrap, audit (Q28) and HTTP (Q29) done (ADR 0012); the keyring backends compile for Windows and macOS but have not run there.
- [x] Q14 Move the API token into the vault. Done: stored as `api-token`, legacy file imported and removed; rotation replaces it (Q30).
- [x] Q28 Vault audit events for set, reveal and delete, in the audit chain (F1-T18); record names and actors only, never values. Done: `Service.Set`, `Reveal` and `Delete` append an event before acting (`vault.set`, `vault.reveal`, `vault.delete`; facts hold the name only); a refused audit write refuses the action (`internal/vault/service_audit_test.go`).
- [x] Q29 Vault HTTP routes: list names/dates, set, delete; full scope; no route returns a value. `api-token` stays reserved for `POST /v1/auth/rotate-token`; tests and OpenAPI lint pass.
- [x] Q30 Rotation of the local `api-token`: a rotate action replaces the vault item and the live token; the old token fails on the next call. Done: `POST /v1/auth/rotate-token` (full scope; `api/openapi.yaml` operationId `rotateToken`), `auth.LocalToken` (`internal/auth/token.go`), `internal/api/auth.go`. The new token is returned once and is never stored in plain text. No separate revocation route: the rotate action revokes the old token, and revoking without a replacement would leave no local client. Tests: `internal/api/auth_test.go`, `internal/auth/token_test.go`; binary smoke (old 401, new 200, both after restart).
- [x] Q31 Master-key rotation at startup: `CALANGO_VAULT_KEY_VERSION`, `CALANGO_VAULT_KEY_PREVIOUS[_FILE]`; `Service.Rotate` re-wraps and audits `vault.rewrap` before serving; startup refuses rows whose version has no key (ADR 0012 amendment). Done; binary smoke in `DEV/VERIFY.md`.

P2:
- [x] Q15 Approvals API validation: `TestApprovalDecisionAPIExecutesApprovedSnapshot` creates a request through the real service, calls the HTTP decision route, and verifies the approved snapshot reaches an injected fake executor. Live Codex/MCP and external connector action remain unverified.
- [x] Q16 Routines: HTTP routes, manual run, `routine_finish`, execution and `routine.fired`/`routine.failed` events are implemented and tested (F1-T09/Q33). Event-based triggers are deferred to phase 3; see `modules/routines.md`.
- [ ] Q17 Memory: `memory_save`, CRUD, budgeted injection and startup/daily purge are implemented. Project scope is out of V1; owner addendum for channel/agent actions as memory context remains a design blocker.
  - Owner addendum (2026-11-23): the intelligence layer is Orquestrador Maestro, and channel and agent actions are also saved and used as the memory context base. Capture rules, storage, retention and deletion design remain open (`DEV/SPECS/modules/memory.md`).
- [ ] Q18 Usage: desktop panel (F1-T17). The panel exists and the fake-run desktop smoke shows an unknown-cost row; known/estimated pricing and a real provider cost remain unverified. Known/estimated split is in `GET /v1/usage`.
- [x] Q19 Pairing: `/v1/pair` rate limit is proxy-aware via `CALANGO_TRUSTED_PROXIES` (F1-T15). Done 2026-11-23; the VPS must set the list (`DEV/DEPLOYMENT.md`).

P3:
- [x] Q20 MCP (F1-T12): go-sdk v1.3.0 (MIT; NOTICE, 2026-11-23). Done: connectors and grants (migration 0013), gateway per turn at `/mcp` with per-turn token in `CALANGO_MCP_TOKEN` (ADR 0018), `read` and `write` scopes, writes through approvals, Codex wiring, `/v1/connectors*` and `/v1/agents/{id}/tools*`. Open: connector credentials; Calango's own tools (Q17 and the rest); restart durability of results; live Codex check with `workspace-write` (owner authorisation for a model call). Reference: `agents/mcp.py` (1040), `services/mcp_connection.py` (3655), `services/mcp_audit.py` (517), `agents/mcp_oauth.py` (676).
- [x] Q21 Screens (F1-T23): agents, conversations, approvals, routines, usage, settings and audit screens are implemented. Fake-provider flows, mocked approval-card dispatch and real HTTP decision integration with a fake executor pass; isolated Settings/Audit smoke and responsive site review pass. Live external action and Phase 1 exit remain separate.

Blocked by D3 (conversations and events):
- [x] Q22 Conversations/runs (F1-T07/T13): routes, steer, persisted replies, idempotency, per-agent one-active-run FIFO and REST state restoration are implemented and tested. WebSocket event replay is intentionally not part of the contract (ADR 0015).
- [x] Q32 Provider chain per agent with fallback and named accounts (ADR 0016). Done 2026-11-23: migration 0010 (`provider_accounts`, `agent_providers`; drops `agents.harness`/`agents.model`), agent `providers`, fallback loop with `provider_attempt` steps, provider accounts CRUD, `codex` and `anthropic` adapters only. Verified by unit, API and migration tests (`DEV/VERIFY.md`); no real model call. Owner decided 2026-11-23.
- [x] Q33 Routines fire an agent run (owner decided 2026-11-23, not a DM message): routes (`/v1/routines*`, including run now and the runs listing), `notify_on`, quiet window, `routine_finish` as the reply's control line (ADR 0017), executor wiring (one DM per routine, one agent run per attempt), `routine.fired` and `routine.failed` events. Done 2026-11-23; verified by unit and API tests and a daemon smoke (`DEV/VERIFY.md`). This is Q16's executor decision. Known limits in `modules/routines.md`.
- [x] Q23 WebSocket `/v1/events`: `github.com/gorilla/websocket` v1.5.3 (BSD-2-Clause, Go 1.22 compatible; ADR 0015); bearer header auth; owner-filtered; no replay (F1-T13). Tests in `internal/api/events_test.go` and `internal/events`; race run and daemon smoke (401 without token) passed.

Maestro:
- [x] Q24 Maestro CLI behavior validated in a throwaway home; Calango manager/API/scheduler, agent linking, audit, update and rollback code now exist. F1-T16 stable Codex home (ADR 0014) is in code and fake-tested; linked agents use it. Real Codex session and the migration policy for earlier per-version logins remain open.

Verification and site:
- [x] Q25 Docker bridge validation. Done 2026-11-25: site and node images built and ran on a user-defined bridge; both `/healthz` endpoints passed; the node database survived container recreation in a named volume. See `DEV/VERIFY.md`.
- [x] Q26 Site reviewed at desktop 1440 px and phone 390 px, in light and dark themes; no horizontal overflow at 390 px. Review found future phone/cloud/team features presented in plan cards; reconcile copy before launch (Q27).
- [ ] Q27 Owner site items: CLA legal name, contacts (`licenca@`, `seguranca@`, `conduta@`), Caddy domain, Terms link and real prices. Also decide how to label phone/cloud/team plan features that are still roadmap items.

## 0. Owner decisions (block parts of V1)

- [x] D1 Calango code will be **open** (decided by the owner on 2026-11-23).
- [x] D1a License: **AGPL-3.0** + commercial license + CLA + trademark policy (ADR 0010). Files created.
- [ ] D1b Legal review of `CLA.md`, `COMMERCIAL.md` and `TRADEMARKS.md`; fill in the maintainer's legal name and the contacts (`licenca@`, `seguranca@`, `conduta@`).
- [x] D2 Remote screen control: **Pro** (owner, 2026-11-23). Update PLANO §14 and PRODUCT_HYBRID to match.
- [x] D3 First MVP harness: **Codex CLI** (owner, 2026-11-23). Sandbox: `workspace-write` chosen by the owner after the risk was shown: Codex runs commands and writes files without Calango approval until the MCP gateway (F1-T12) serves its tools. Other harnesses in V1: Claude Code CLI, OpenCode, Antigravity (Google, CLI `agy`; add to the harness spec), direct API. → `SPECS/modules/harness.md`
- [x] D4 Domain and Go module name: **keep `calango`** (owner, 2026-11-23). Domain still to confirm with Q27.
- [ ] ⛔ D5 Research completed (2026-11-23): `tsnet` is BSD-3-Clause; official Tailscale Terms §2.3 restrict commercial exploitation/resale and do not expressly resolve embedding in Calango; Personal plan limits are recorded above and in ADR 0005. Owner must choose written legal/vendor clearance or another transport before F2-T02.
- [x] D6 Maestro automatic update: **on by default** (owner, 2026-11-23). ADR 0009 amendment written; Q24 unblocked.

---

## Phase 0 — Foundation

- [x] Go 1.27.4 installed in `~/.local/opt/go` (SHA-256 verified against go.dev).
- [x] `go vet ./...` clean.
- [x] `go test ./...` passes.
- [x] `gofmt -l` with no output.
- [x] `calango` starts and `GET /healthz` returns 200; POST returns 405.
- [x] F0-T04 CI on GitHub Actions: vet, test, build linux/windows/darwin on amd64 and arm64. Six local builds passed; GitHub run 45188079827 on `dee8efa` succeeded. Current uncommitted changes are not covered by that run.
- [x] F0-T05 `internal/config`: reads `CALANGO_*` in one place and validates at startup. `CALANGO_ADDR` keeps the loopback default and validates host/port before the scheduler.
- [x] F0-T06 Semantic validation of OpenAPI in CI with Redocly CLI 2.81.0 pinned; current local lint is valid with one non-blocking warning.
- [x] F0-T07 Deterministic generator from `tokens.json` to CSS and TypeScript/Dart constants; CI fails if the generated files are missing or drift. The i18n catalog generator comes with F1-T26.
- [x] Token contract validation: `node scripts/generate-tokens.mjs --check` confirms the three generated artifacts. Flutter composes `ThemeData` from the Dart tokens in F2-T03.

---

## Phase 1 — Local desktop MVP (Windows first)

### Core
- [x] F1-T01 `internal/store`: SQLite (`modernc.org/sqlite`), WAL, immutable migration runner, in-memory tests. → DATA_MODEL, ADR 0002
- [x] F1-T02 Initial migrations: agents, conversations, messages, runs, run_steps, usage, audit_log.
- [x] F1-T03 `internal/gateway`: `Provider` interface, budget check before the call, usage after, fake provider. → `modules/gateway.md`, ADR 0007
- [x] F1-T04 First provider: direct Anthropic API (HTTP with streaming); key comes from the vault. Done in `internal/anthropic`; agents reach it through the provider chain (Q32, ADR 0016). Only fake-provider runs were exercised here; no real model call was authorized.
- [ ] F1-T05 `internal/harness`: Codex CLI adapter, per-account `CODEX_HOME`, daemon wiring and fake-CLI E2E are implemented. No authorized real model call; Codex `--json` usage schema and other V1 harnesses remain open.
- [x] F1-T06 Agents: CRUD, global pause, monthly limit (service and store; HTTP wiring in F1-T13/T15, pause enforcement on runs in F1-T07). → `modules/agents.md`
- [x] F1-T07 Conversations/runs: persisted/streamed steps, stop, steer, retry, crash recovery, idempotency, provider fallback, per-agent one-active-run FIFO and REST state restoration are implemented and tested. WebSocket event replay is intentionally omitted by ADR 0015.
- [ ] F1-T08 Approvals: canonical snapshot/digest, decide/receipt, always-allow, expiry, `waiting_approval` and reconciliation are tested. API integration exercises service request + HTTP decision with a fake executor; mocked renderer dispatch passes. Live MCP/provider action remains unverified.
- [x] F1-T09 Persisted routines: cron + interval, occurrence ID, idempotent retries, notify_on and quiet window. → `modules/routines.md`; executor, HTTP, events, manual run and `routine_finish` done (Q33). Event triggers are phase 3. Desktop fake-provider E2E verifies routine create and pause/resume.
- [ ] F1-T10 Vault: envelope, store, service, bootstrap, audit, HTTP, token/master-key rotation done. OS keyring backend compiles for Windows/macOS/Linux but has not run on Windows or macOS.
- [ ] F1-T11 Memory: save/list/edit/delete, budgeted injection, `memory_save` and startup/daily purge implemented. Project scope removed from V1; owner addendum remains open.
- [x] F1-T12 MCP: connect a server, local MCP gateway per turn, scopes none/read/write. → `modules/tools-mcp.md`; connectors, per-turn gateway, grants and approval-gated writes are implemented (ADR 0018). Connector credentials, other built-in tools, result restart durability and live Codex validation remain open.
- [x] F1-T13 API v1 and WebSocket routes are served and tested; approval request/HTTP decision integration uses a fake executor. Reconnect reloads persisted state through REST; WebSocket replay is intentionally omitted by ADR 0015.
- [x] F1-T14 Local desktop token, stored in the vault. Done: token is the vault item `api-token`, created on first start; a legacy `DataDir/api-token` file is imported and removed; every `/v1` route requires it. Rotation done (Q30); there is no revocation route without a replacement, by design.
- [ ] F1-T15 Device pairing and authentication (local version, no phone yet). → `modules/pairing-auth.md`. Partial: local live-daemon acceptance passed 2026-11-25 for code redemption, single use, token revocation, rate limit and `full`/`read`/`approve_only` scopes (see `DEV/VERIFY.md`, #58). Mobile QR, remote transport and the VPS administrator password remain phase-gated.
- [ ] F1-T16 Maestro integration: manager, isolated install/verify, link/unlink, V1 update check, default auto-update, audit, rollback and status UI are implemented and fake-tested. The Codex home is the stable `<calango-home>/maestro/.codex` (ADR 0014): each release is verified in its staging home, then activated into the stable home, so sign-in survives updates and rollbacks. Partial: no real Codex session; earlier per-version logins are not migrated (owner policy pending).
- [ ] F1-T17 Usage/cost panel per agent, routine, project, model and node; REST and desktop show known/estimated/unknown separately. Real provider pricing/cost remains unverified; project grouping has no projects table in V1.
- [ ] F1-T18 Append-only audit: hash chain, verify/export API and desktop screen are implemented. Full remote-action and permission-event coverage remains unverified; no remote-node action is in Phase 1.

### Desktop (Electron)
- [x] F1-T20 `apps/desktop`: Electron + Vite + React + TS strict; `contextIsolation`, `sandbox`, strict CSP. Built and launched on Linux; Windows remains unverified.
- [x] F1-T21 Main process starts and watches `calango`; restarts with backoff and reports state. Desktop E2E passed; child PATH filtering prevents npm's `node_modules/.bin` from shadowing Codex.
- [x] F1-T22 `packages/sdk-ts` generated from OpenAPI; `packages/tokens` generating CSS; typed client used by the renderer.
- [x] F1-T23 Screens: conversations, steps, approval card, routines, agents, usage, settings. Screens are implemented; fake-provider flows and mocked approval-card dispatch pass. Isolated Electron smoke exercised Settings and Audit; live approval action and full Phase 1 acceptance are separate gates.
- [ ] F1-T24 Tray, start with the system, global shortcut for quick conversation. Code is wired; Linux startup entry create/remove passed an isolated UI smoke. Tray creation and shortcut registration ran without warnings, but OS-level hotkey activation and Windows/macOS behavior remain unverified.
- [ ] F1-T25 Signed Windows NSIS installer + signed auto-update; Linux `.deb`; notarized macOS DMG; package `NOTICE`. Requires signing credentials and platform packaging/verification.
- [x] F1-T26 i18n pt-BR, en and es; single catalog with deterministic generation and CI drift check. `node scripts/generate-i18n.mjs --check` passes.

### Phase 1 exit criterion
- [ ] Phase 1 exit: install on Windows, talk to an agent, create a daily routine, approve an action, and see cost in the panel. Not met: Windows install/signing, authorized live approval/action, real provider cost and cross-platform acceptance remain unverified.

---

## Remaining unchecked work — triage (2026-11-23)

- **Owner-blocked:** Q03/D5 transport/legal clearance; Q06 permission to delete `/tmp/agenticos`; D1b/Q27 legal identity, terms, prices, domain and plan copy; Q17 memory capture/retention; Q18/F1-T17 verified pricing; live model/MCP and external approval authorization (F1-T04/T05/T08/T12); F1-T16 profile-migration policy; personal-data export/deletion scope; authorization for the next push.
- **Platform-blocked:** Q13/F1-T10 Windows/macOS keyring; F1-T24 OS shortcut/tray and cross-platform login; F1-T25 signing/install/update credentials and platform verification; Q27 site hosting; F2-T07 store review.
- **Implementable engineering:** F1-T05 JSONL usage parsing and other V1 harnesses; F1-T12 connector credentials, remaining built-ins and result durability; F1-T15 local QR; F1-T18 applicable audit coverage; database backup/restore; first-use/privacy/Maestro docs; desktop/site accessibility; reference-code/license review; site icon. These remain unchecked until implemented and verified.
- **Implementable but phase-gated:** F1-T15 mobile/VPS parts and F1-T17 project grouping; all F2–F6 tasks remain behind their preceding phase exit. F7 is post-V1. WebSocket replay is intentionally omitted; REST reload meets the reconnect contract (ADR 0015).

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
- [ ] F3-T08 Automatic Maestro update (on by default, D6; the owner can turn it off) with rollback and `verify`. → `modules/maestro.md`
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

- [x] Secret audit: canary tests cover run_steps, logs and API responses (`internal/conversations`, `internal/api`).
- [x] API limits: bounded request bodies and proxy-aware `/v1/pair` rate limiting are tested.
- [ ] Database backup and restore: offline runbook in `BACKUP_RESTORE.md` and pre-migration snapshot recovery test are implemented; previous-binary compatibility and OS-keyring restoration remain unverified.
- [ ] Export and deletion of personal data exercised; first run detects node state, model, harness, vault and connector without leaking content.
- [x] Telemetry only with opt-in, without content: Calango ships no external telemetry/analytics client; Maestro CLI subprocesses force `ORQUESTRADOR_MAESTRO_TELEMETRY=0` and an app-private `XDG_CONFIG_HOME` (ADR 0014). Code search and Go tests show no Calango telemetry sender.
- [ ] User documentation: first use, pairing, VPS, Maestro, privacy.
- [ ] Accessibility review (keyboard, screen reader, contrast) on desktop, mobile and the site.
- [x] Site: "Contribute to V1" section and footer with AGPL-3.0 (done 2026-11-23).
- [ ] Site: replace "coming soon" with real prices once defined; host it; simplified icon for small sizes (`DEV/BRAND/BRAND.md`).
- [x] Publish the repository at `IAPro-Community/calango`. Owner-authorized push and GitHub CI run 45188079827 on `dee8efa` are recorded in Q04/Q05; current uncommitted changes are not published.
- [ ] Validate `docker build` and `docker run` of the site `Dockerfile` and `deploy/Dockerfile.node` (blocked: Docker daemon unavailable in this environment; the user needs to grant daemon access).
- [x] Validate the `.github/workflows/ci.yml` workflow after the first push. GitHub run 45188079827 on `dee8efa` succeeded; current uncommitted changes are not covered.
- [x] Full site visual review at desktop 1440 px and mobile 390 px, light/dark. No mobile horizontal overflow; future-feature claims in plan cards are recorded under Q27.
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
