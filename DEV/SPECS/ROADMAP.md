# Roadmap with tasks

A [sequência operacional por ondas](../ONDAS_DE_ENTREGA.md) define os gates vigentes. Na v0.2.0, styleguide e núcleo liberam Web (#188), Web libera Desktop Windows (#41) e Windows libera mobile (#71). As fases numeradas abaixo preservam a referência dos IDs originais.

Each task has an ID (`F<phase>-T<nn>`), a module, dependencies and a done criterion. Mark `[x]` when finished and record it in `WORKLOG.md`. Do not start a phase before the previous one meets its exit criterion.

---

## Phase 0 — Foundation

Exit: `go vet ./...` and `go test ./...` pass; `calango` answers `/healthz`; invalid configuration fails before the server starts; contracts and supported builds are verified by CI.

- [x] F0-T01 Repo skeleton, `go.mod`, `cmd/calango`, `internal/core`, `internal/api`.
- [x] F0-T02 `DEV/` docs, ADRs, OpenAPI draft, tokens.
- [x] F0-T03 Install Go 1.22+ and run `go vet` / `go test`. Done with Go 1.27.4 in `~/.local/opt/go`; vet, tests and healthz pass.
- [x] F0-T04 CI (GitHub Actions): vet, test, build linux/windows/darwin on amd64+arm64. Workflow and six local builds verified; GitHub run 45188079827 on `dee8efa` passed. Current uncommitted changes are not in that run.
- [x] F0-T05 `internal/config`: reads `CALANGO_*` variables in one place, validates at startup.
- [x] F0-T06 OpenAPI semantic lint in CI (Redocly CLI pinned); current local lint is valid with one non-blocking warning.
- [x] F0-T07 Design token generator for CSS and TypeScript/Dart constants, with drift checking in CI. i18n is in F1-T26, together with the catalog creation.

## Phase 1 — Local desktop MVP

Exit: on Windows, install the app, talk to an agent, create a daily routine, receive and approve an approval card, see the cost in the usage panel.
- Current gate: not met. Linux fake-Codex E2E verifies agent setup, chat reply, routine and usage. Approval service request + HTTP decision with an injected fake executor and mocked card dispatch are tested; live MCP/provider action remains unverified. Windows packaging, verified provider cost, and platform acceptance remain open.

### Core
- [x] F1-T01 `internal/store`: SQLite (modernc), WAL, migration runner. In-memory tests. → `DATA_MODEL.md`
- [x] F1-T02 Initial migrations: agents, conversations, messages, runs, run_steps, usage, audit_log. (dep: T01)
- [x] F1-T03 `internal/gateway`: `Provider` interface, budget before, usage after, fake provider for tests. → `modules/gateway.md` (dep: T02)
- [x] F1-T04 First provider: direct Anthropic API (HTTP, streaming). Key from the vault. (dep: T03, T10) Done in `internal/anthropic`; agents select it through their provider chain (Q32, ADR 0016).
- [ ] F1-T05 `internal/harness`: harness interface + adapter. Owner decision (2026-11-23): first adapter is Codex CLI; then Claude Code CLI, OpenCode, Antigravity (Google, CLI) and direct API. → `modules/harness.md` (dep: T03). Partial: Codex CLI provider, per-account `CODEX_HOME`, daemon wiring and fake-CLI desktop E2E; no real model call is authorized, `--json` schema and other harnesses remain open.
- [x] F1-T06 Agents: CRUD, global pause, monthly budget (service and store; HTTP in F1-T13/T15, pause enforcement on runs in F1-T07). → `modules/agents.md` (dep: T02)
- [x] F1-T07 Conversations and runs: DM, persisted/streamed steps, stop/steer, recovery and deduplication. → `modules/conversations.md` (dep: T03, T06). Done: per-agent one-active-run FIFO, durable assistant messages and REST state reload are tested; WebSocket event replay is intentionally omitted by ADR 0015.
- [ ] F1-T08 Approvals: immutable canonical action bound by digest, decision, separate receipt and "always allow" rules. → `modules/approvals.md` (dep: T07); source ADR 0011 (P2). Partial: domain/store/service/HTTP, `waiting_approval`, startup reconciliation and API integration with a fake executor are tested; mocked renderer covers card dispatch. Live MCP/provider action remains unverified.
- [x] F1-T09 Persisted routines: cron + interval, occurrence ID, idempotent retries, notify_on and quiet window. → `modules/routines.md` (dep: T07); source ADR 0011 (P2). Done for cron and interval (Q33): coalescing, retries with backoff, quiet window, pause, restart, HTTP, events, and an executor that fires one agent run per attempt (ADR 0017), with finish through the control line. Event triggers are phase 3.
- [ ] F1-T10 Vault: Windows Credential Manager, macOS Keychain, Linux Secret Service. → `modules/vault.md`; source ADR 0011 (P1). Partial: envelope, store, service, bootstrap, audit, HTTP and token/master-key rotation done; OS keyring backends compile but Windows/macOS runtime is unverified.
- [ ] F1-T11 Memory: save/list/edit/delete and injection under a token budget. → `modules/memory.md` (dep: T07); service/store/HTTP/agent-scoped `memory_save` and startup/daily purge are implemented. Project scope is removed from V1; owner addendum for channel/agent action memory remains open.
- [x] F1-T12 Tools via MCP: connect an MCP server, scopes per agent. → `modules/tools-mcp.md` (dep: T07, T08); source ADR 0011 (P3). Done (2026-11-23, ADR 0018): connectors, grants `read`/`write`, per-turn gateway at `/mcp`, writes through approvals. Open: connector credentials, Calango's own tools other than `memory_save`, live Codex check.
- [x] F1-T13 API v1 per `api/openapi.yaml` + WebSocket `/v1/events`. REST and WebSocket routes are served and tested; API approval request/decision integration uses a fake executor. Reconnect reloads persisted state through REST; WebSocket replay is intentionally omitted by ADR 0015.
- [x] F1-T14 Local auth: desktop app token generated on first start, stored in the vault. (dep: T10) Done: token is the vault item `api-token`, created on first start; a legacy `DataDir/api-token` file is imported and removed; every `/v1` route requires it; `POST /v1/auth/rotate-token` replaces it at once (Q30).
- [ ] F1-T15 Local device pairing/authentication, revocation and scopes. → `modules/pairing-auth.md` (dep: T10). Partial: pairing codes, device tokens, device list, revocation and scopes `full`/`approve_only`/`read` are implemented; local live-daemon acceptance passed 2026-11-25 (see `../VERIFY.md`, #58). The per-origin rate limit is proxy-aware via `CALANGO_TRUSTED_PROXIES` (Q19). Mobile QR, remote transport and the VPS administrator password remain phase-gated.
- [ ] F1-T16 Orquestrador Maestro integration (V1 line, npm). → `modules/maestro.md`, ADR 0009 (dep: T05, T13). Partial: manager/API/scheduler, isolated per-version install/verify, linking, audit, update, rollback and status UI are implemented and fake-tested. `CODEX_HOME` is version-specific, contrary to ADR 0014's stable-home decision; no real Codex session has been run.
- [ ] F1-T17 Usage/cost panel per agent, routine, project, model and node; distinguish real, estimated and unknown cost. → ADR 0007, `DATA_MODEL.md` (dep: T03, T07); source ADR 0011 (P2, `services/usage_report.py`). Partial: REST groups by agent, routine, project, node or model; the desktop panel shows known, estimated and unknown rows and passed a fake-run smoke. Real provider pricing is not configured, so no actual cost has been verified; the project group has no projects table in V1.
- [ ] F1-T18 Append-only audit of approvals, receipts, actions and permission changes, without secrets or unnecessary content. Hash chain, checkpoint, verification/export API and desktop audit UI are tested; full event coverage for remote actions remains unverified.

### Desktop
- [x] F1-T20 `apps/desktop`: Electron + Vite + React + TS strict, `contextIsolation`, sandbox and strict CSP. Built and launched on Linux; Windows remains unverified. → `modules/desktop-electron.md`
- [x] F1-T21 Main process starts and watches `calango`, restarts with backoff and reports state. Desktop E2E passed; child PATH filtering prevents npm's `node_modules/.bin` from shadowing the user's Codex CLI.
- [x] F1-T22 `packages/sdk-ts` generated from OpenAPI; `packages/tokens` generates CSS. Typed client is used by the renderer.
- [x] F1-T23 Screens: DMs, conversation with steps, approval card, routines, agents, usage and settings. All screens are implemented; fake-provider flows and mocked approval-card dispatch pass. Isolated Electron smoke exercised Settings and Audit; live action and Phase 1 acceptance are separate gates. Source: ADR 0011 (P3, screen reference only).
- [ ] F1-T24 Tray, start with the system, global shortcut. Code is wired; Linux startup entry create/remove passed an isolated UI smoke. Tray creation and shortcut registration ran without warnings, but OS-level hotkey activation and Windows/macOS behavior remain unverified.
- [ ] F1-T25 Signed Windows NSIS installer + auto-update; Linux `.deb`; notarized macOS DMG.
- [x] F1-T26 i18n pt-BR, en, es from the first screen; one catalog with deterministic generation and drift checking in CI. `node scripts/generate-i18n.mjs --check` passes.

## Phase 2 — Phone

Exit: away from home, approve on the phone an action requested by an agent on the PC, and receive a push.

- [ ] F2-T01 Pairing: single-use code, QR, exchange for a device token, revocation. → `modules/pairing-auth.md`
- [ ] F2-T02 Tailscale transport (evaluate embedded `tsnet`) behind `internal/transport`. → ADR 0005
- [ ] F2-T03 `packages/sdk-dart` generated; `ThemeData` composed from the Dart constants generated from tokens.
- [ ] F2-T04 `apps/mobile` Flutter: pair, nodes, conversations, approvals, routines, tasks. → `modules/mobile-flutter.md`
- [ ] F2-T05 Minimal Calango Cloud: accounts and devices. → `modules/cloud.md`
- [ ] F2-T06 Push relay (APNs/FCM), only "there is news". → `modules/push.md`
- [ ] F2-T07 TestFlight and Play internal testing.

## Phase 3 — VPS and first revenue

Exit: with the PC off, a routine with `run_on=auto` runs on the VPS; the Pro subscription is charged and enables the VPS node.

- [ ] F3-T01 Official multi-arch Docker image + one-line installer. → `DEPLOYMENT.md`
- [ ] F3-T02 Event log and replication between nodes. → `modules/nodes-sync.md`, ADR 0008
- [ ] F3-T03 Hybrid execution: `run_on` pc/vps/auto, owner handover when the PC goes down.
- [ ] F3-T04 Ed25519-signed license, offline validation, tolerance, downgrade. → `modules/licensing-billing.md`
- [ ] F3-T05 Billing: Stripe (web/desktop) + store purchases; single plan source in the Cloud.
- [ ] F3-T06 Plan limits enforced in the core (`Entitlements`).
- [ ] F3-T07 Interface shows an offline node: which one, since when, what is waiting for it.

## Phase 4 — ADE

Exit: from a task, an agent creates a worktree, changes code, runs tests and opens a PR, on the PC or on the VPS.

- [ ] F4-T01 Projects: register repo, node, agents, `DEV/` memory. → `modules/ade.md`
- [ ] F4-T02 Worktree per task with diff, commands and test results linked to the run; cleanup when finished.
- [ ] F4-T03 Terminal: PTY in `calango` (ConPTY on Windows) over WebSocket; xterm.js in the desktop app.
- [ ] F4-T04 Monaco editor, diff, file tree.
- [ ] F4-T05 GitHub: branch, PR, CI status and reviewable run evidence.
- [ ] F4-T06 Preview server opened in the internal browser.
- [ ] F4-T07 Import skills from Claude Code and Codex.

## Phase 5 — Browser and computer use

Exit: the agent fills in a form in the internal browser; the person takes over mid-way and hands it back.

- [ ] F5-T01 Tabs with `WebContentsView`, profiles per agent. → `modules/browser.md`
- [ ] F5-T02 The agent drives the tab through CDP; take over/hand back without reloading.
- [ ] F5-T03 Barrier detection (robot, login, empty page) with a warning to the agent.
- [ ] F5-T04 Headless browser on the VPS (chromedp) with the same interface.
- [ ] F5-T05 Computer use (desktop apps), off by default. → `modules/computer-use.md`
- [ ] F5-T06 Screen control from the phone (paid), with notice and audit.

## Phase 6 — Channels and teams

Exit: two members and three agents talk in a channel; the router picks who answers.

- [ ] F6-T01 Channels, threads, mentions. → `modules/channels-teams.md`
- [ ] F6-T02 Router (cheap model, 0–2 agents, never grants permissions).
- [ ] F6-T03 Members, roles, per-channel permissions, delegated approval.
- [ ] F6-T04 Team plan in billing.

## Phase 7 — Expansion

- [ ] F7-T01 Own relay (replaces Tailscale as an option).
- [ ] F7-T02 External channels: Telegram, WhatsApp, Slack, email.
- [ ] F7-T03 Voice: dictation and spoken replies.
- [ ] F7-T04 Files and documents with permissions.
- [ ] F7-T05 Marketplace of routines and agents.
