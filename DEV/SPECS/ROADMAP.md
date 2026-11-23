# Roadmap with tasks

Each task has an ID (`F<phase>-T<nn>`), a module, dependencies and a done criterion. Mark `[x]` when finished and record it in `WORKLOG.md`. Do not start a phase before the previous one meets its exit criterion.

---

## Phase 0 — Foundation

Exit: `go vet ./...` and `go test ./...` pass; `calango` answers `/healthz`; invalid configuration fails before the server starts; contracts and supported builds are verified by CI.

- [x] F0-T01 Repo skeleton, `go.mod`, `cmd/calango`, `internal/core`, `internal/api`.
- [x] F0-T02 `DEV/` docs, ADRs, OpenAPI draft, tokens.
- [x] F0-T03 Install Go 1.22+ and run `go vet` / `go test`. Done with Go 1.27.4 in `~/.local/opt/go`; vet, tests and healthz pass.
- [x] F0-T04 CI (GitHub Actions): vet, test, build for linux/windows/darwin on amd64+arm64. Workflow updated and six local builds verified; GitHub run still pending.
- [x] F0-T05 `internal/config`: reads `CALANGO_*` variables in one place, validates at startup.
- [x] F0-T06 OpenAPI semantic lint in CI (Redocly CLI pinned); local lint passed with no problems or warnings.
- [x] F0-T07 Design token generator for CSS and TypeScript/Dart constants, with drift checking in CI. i18n is in F1-T26, together with the catalog creation.

## Phase 1 — Local desktop MVP

Exit: on Windows, install the app, talk to an agent, create a daily routine, receive and approve an approval card, see the cost in the usage panel.

### Core
- [ ] F1-T01 `internal/store`: SQLite (modernc), WAL, migration runner. In-memory tests. → `DATA_MODEL.md`
- [ ] F1-T02 Initial migrations: agents, conversations, messages, runs, run_steps, usage, audit_log. (dep: T01)
- [ ] F1-T03 `internal/gateway`: `Provider` interface, budget before, usage after, fake provider for tests. → `modules/gateway.md` (dep: T02)
- [ ] F1-T04 First provider: direct Anthropic API (HTTP, streaming). Key from the vault. (dep: T03, T10)
- [ ] F1-T05 `internal/harness`: harness interface + adapter for Claude Code CLI. → `modules/harness.md` (dep: T03)
- [ ] F1-T06 Agents: CRUD, global pause, monthly budget. → `modules/agents.md` (dep: T02)
- [ ] F1-T07 Conversations and runs: DM, persisted/streamed steps, stop/guide, recover after reconnection and deduplicate resends. → `modules/conversations.md` (dep: T03, T06)
- [ ] F1-T08 Approvals: immutable canonical action bound by digest, decision, separate receipt and "always allow" rules. → `modules/approvals.md` (dep: T07)
- [ ] F1-T09 Persisted routines: cron + interval, occurrence ID, idempotent retries, notify_on and quiet window. → `modules/routines.md` (dep: T07)
- [ ] F1-T10 Vault: Windows Credential Manager, macOS Keychain, Linux Secret Service. → `modules/vault.md`
- [ ] F1-T11 Memory: save, list, edit, delete; injection into the run context. → `modules/memory.md` (dep: T07)
- [ ] F1-T12 Tools via MCP: connect an MCP server, scopes per agent. → `modules/tools-mcp.md` (dep: T07, T08)
- [ ] F1-T13 API v1 per `api/openapi.yaml` + WebSocket `/v1/events`. (dep: T06–T09)
- [ ] F1-T14 Local auth: desktop app token generated on first start, stored in the vault. (dep: T10)
- [ ] F1-T15 Local device pairing/authentication, revocation and scopes. → `modules/pairing-auth.md` (dep: T10)
- [ ] F1-T16 Orquestrador Maestro integration (V1 line, npm). → `modules/maestro.md`, ADR 0009 (dep: T05, T13)
- [ ] F1-T17 Usage/cost panel per agent, routine, project, model and node; distinguish real, estimated and unknown cost. → ADR 0007, `DATA_MODEL.md` (dep: T03, T07)
- [ ] F1-T18 Append-only audit of approvals, receipts, actions and permission changes, without secrets or unnecessary content. → `SECURITY.md` (dep: T02, T08)

### Desktop
- [ ] F1-T20 `apps/desktop`: Electron + Vite + React + TS strict. → `modules/desktop-electron.md`
- [ ] F1-T21 Main process starts and watches `calango`; restarts if it dies; shows state. (dep: T20)
- [ ] F1-T22 `packages/sdk-ts` generated from OpenAPI; `packages/tokens` generating CSS. (dep: T13)
- [ ] F1-T23 Screens: DMs, conversation with steps, approval card, routines, agents, usage, settings. (dep: T21, T22)
- [ ] F1-T24 Tray, start with the system, global shortcut.
- [ ] F1-T25 Signed Windows NSIS installer + auto-update; Linux `.deb`; notarized macOS DMG.
- [ ] F1-T26 i18n pt-BR, en, es from the first screen; single catalog with deterministic generation and drift checking in CI.

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
