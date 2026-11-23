# Calango — full software plan

Status: planning. Version 1, 2026-11-23. Nothing beyond the foundation (phase 0) is implemented.
Supporting documents: [ARCHITECTURE.md](ARCHITECTURE.md), [PRODUCT_HYBRID.md](PRODUCT_HYBRID.md), [MONETIZATION.md](MONETIZATION.md), [DEPLOYMENT.md](DEPLOYMENT.md).

---

## 1. Vision

**Calango is a personal agent that lives on your computer, extends to your VPS and answers in your pocket.**

It talks, remembers, executes tasks, runs routines, browses the web, builds projects and works in channels with people and other agents. Everything runs on machines the person owns. Calango (the company) provides only identity, licensing, push and updates.

The competitive landscape has changed: self-hosted products, agent desktops and hosted services already offer parts of PC, server, phone, browser, routines and programming. Therefore, **PC + VPS + phone is not an exclusive advantage**. The differentiation hypothesis is to keep one shared contract of identity, budget, policy, approval and evidence when running on the person's own PC and VPS. This hypothesis still has to be demonstrated in V1. Comparables, sources and limits of the research are in [COMPETITIVE_LANDSCAPE.md](COMPETITIVE_LANDSCAPE.md).

## 2. Principles

1. **The nodes belong to the person.** Data, keys and execution stay on their PC and VPS.
2. **One core.** The same Go binary (`calango`) runs on the PC and on the VPS. Clients do not run agents.
3. **Nothing sensitive without approval.** Sending, deleting, publishing, paying, deploying: always an approval card.
4. **All cost is visible.** Every model call is measured, has an owner and has a limit.
5. **Silence is a valid answer.** A routine with no news does not notify anyone.
6. **Original code.** Tucano and agentes.chat are read-only references.

---

## 3. Topology

```
                        ┌──────────────── Calango Cloud (ours) ────────────────┐
                        │ accounts · licenses · billing · push relay · updates  │
                        └───────────────▲──────────────────▲────────────────────┘
                                        │ metadata only    │
          Tailscale (person's tailnet)  │                  │
   ┌────────────────────────────────────┼──────────────────┼───────────────┐
   │                                    │                  │               │
┌──┴────────────────┐          ┌────────┴─────────┐   ┌────┴─────┐   ┌─────┴────┐
│ PC node (free)    │◄────────►│ VPS node (paid)  │   │ iOS      │   │ Android  │
│ Electron + calango│  events  │ Docker calango   │   └──────────┘   └──────────┘
│ files, apps,      │  + git   │ 24/7, routines,  │        ▲ remote control ▲
│ local browser     │          │ projects, browser│        └── of PC and VPS ┘
└───────────────────┘          └──────────────────┘
```

- **PC node**: Electron app that starts `calango` as a child process.
- **VPS node**: `calango` in Docker, with no UI of its own; accessed by the clients and the web.
- **Clients**: desktop (Electron), iOS, Android, web. All speak the same API.
- **Transport**: Tailscale in the initial phase (decided). Own relay later.
- **Calango Cloud**: never sees conversations, files or model keys.

---

## 4. Stack by component

### 4.1 `calango` — core (Go)

| Function | Choice |
|---|---|
| Language | Go 1.22+ |
| API | HTTP + WebSocket, OpenAPI contract (generates the TypeScript SDK) |
| Database | pure-Go SQLite (`modernc.org/sqlite`), versioned and immutable migrations |
| Scheduler | persisted routines (`internal/routines`): cron, intervals, quiet windows and retries; events in phase 3 |
| Model gateway | every token goes through it: budget before, cost after |
| Tool gateway | MCP; each agent sees only its own tools and scopes |
| Harnesses | adapters for agent CLIs (Claude Code, Codex, Gemini, Grok, OpenCode) and direct API |
| Agent browser | Chromium via the DevTools Protocol (`chromedp`) |
| Terminal | PTY in the daemon (`creack/pty`; ConPTY on Windows), streamed over WebSocket |
| Git / ADE | system git; one worktree per task |
| Isolation | Docker on the VPS; on the PC, a restricted working folder + approval per command |
| Network | Tailscale (evaluate embedded `tsnet`) |
| Secrets | OS vault: Windows Credential Manager/DPAPI, macOS Keychain, Linux Secret Service |

### 4.2 Desktop — Electron (Windows, macOS, Linux)

- **Main process (Node)**: window, tray, global shortcuts, auto-update, starts and watches `calango`.
- **Renderer (React + TypeScript + Vite)**: the entire interface. Talks to `calango` through the API, just like the phone.
- **Internal browser**: real tabs with `WebContentsView`. The agent drives the same tab through the Electron debugging protocol, and the person can **take over** the tab at any time (and hand it back).
- **ADE editor**: Monaco. **Terminal**: xterm.js connected to the daemon's PTY.
- **Windows**: signed NSIS installer (code-signing certificate from the first public release; Tucano suffers from SmartScreen warnings because it does not sign), starts with the system, tray icon.
- **macOS**: signed and notarized. **Linux**: `.deb` and AppImage (AppImage is restricted on Ubuntu 24.04+, so `.deb` is the main path there).

### 4.3 Mobile — iOS and Android

- **Flutter** (decided 2026-11-23). One Dart codebase for iOS and Android.
- Because the desktop is React/TypeScript, sharing happens through a **contract**, not through code:
  - **API**: the Calango OpenAPI generates two clients, TypeScript (desktop/web) and Dart (mobile). No client is written by hand.
  - **Design**: tokens (colors, typography, spacing) in a single JSON, which generates CSS for the desktop and `ThemeData` for Flutter.
  - **Translations**: a single catalog that generates the Flutter ARB files and the desktop JSON.
- Cost of this choice: duplicated screens (React and Flutter). Mitigation: the mobile app has a smaller, focused scope (remote control, approvals, conversations) and does not replicate the ADE.
- Features: conversations, approvals, routines, tasks, notifications, node status, remote control of the PC and the VPS.
- Push through our relay (APNs/FCM), only with "there is news"; the app fetches the content from the node.
- Purchase: the stores' systems (via RevenueCat or direct integration), with the subscription also recognized on the desktop.

### 4.4 Web

The same React app, served by `calango` itself. Used to access the VPS from any browser.

### 4.5 Calango Cloud (Go, ours)

| Service | Function |
|---|---|
| Account | login, devices, list of the person's nodes |
| Licenses | issues an Ed25519-signed key that the VPS node validates offline |
| Billing | Stripe (web/desktop) + store purchases; a single source of truth for the plan |
| Push relay | forwards notices to APNs/FCM, without content |
| Updates | stable/beta channels for the desktop and the Docker image |
| Telemetry | only with opt-in, without content |

---

## 5. Functional modules

### 5.1 Personal assistant
- Text and voice chat (dictation and spoken replies in a later phase).
- **Configurable daily briefing**: calendar, important emails, tasks, what the routines found.
- **Proactivity**: the agent suggests next steps in a checkbox card, without blocking.
- "For you": a single inbox with pending questions, approvals and deliveries.

### 5.2 Agents
- Several agents per account, each with a role, instructions, model/harness, tools, a monthly budget and where it runs (PC, VPS, automatic).
- Creation assistant: describe the agent, and it drafts it.
- Parallel replies with a configurable limit; guide a reply in progress; stop only one.
- Global pause of all agents.

### 5.3 Channels and DMs
- DM with each agent; channels with several agents (and people, on the Team plan).
- **Router**: a cheap model picks who replies (0 to 2 agents). It never grants permissions.
- Threads, mentions, quoting a passage.
- The router's choice appears at the top of the reply, not as a loose line in the channel.

### 5.4 Routines
- Triggers: schedule (cron), interval, connector event (new email, PR opened, webhook).
- Destination: PC, VPS or automatic.
- **Retry and alert** on failure (agentes.chat loses weekly deliveries because it does not do this).
- Quiet window; a routine with no news does not notify.
- Routine calendar and run history with cost.

### 5.5 Tasks (Board)
- Task kanban: mine, suggested by agents, in progress by agents.
- A task can span several executions until it finishes.
- Routines stay off the Board (they are a schedule, not tasks).

### 5.6 Approvals
- Card with the exact action, target and arguments.
- Receipt afterwards: completed, failed (with reason) or not run.
- Levels per agent: always ask, ask only for sensitive actions, always approve that type.
- Approval from the phone and, on Team, by a substitute.
- Approvals respect the order in which they were requested.

### 5.7 Internal browser
- Tabs, persistent profiles per agent, ad blocker and cookie-notice blocker.
- The agent identifies **barriers** (robot check, login, empty page) and reports them instead of insisting.
- The person takes over and hands back the tab without reloading.
- Page content is always treated as data, never as instructions (defense against prompt injection).

### 5.8 ADE — agent-driven development environment
- Separate **projects**, each with its repository, agents, connectors and memory (`DEV/` as the project's memory).
- Each task opens its own **worktree**: several agents work in parallel without stepping on each other.
- Editor, terminal, diff, running tests, preview server opened in the internal browser.
- GitHub/GitLab integration: branch, PR, review, CI.
- A project can run on the PC or the VPS; synchronization is always through git.
- Import skills from Claude Code and Codex.

### 5.9 Computer use (PC)
- The agent operates desktop apps (clicks, keyboard, screenshots). Off by default; turned on per app.
- Control of the PC screen from the phone (paid plan), with a visible notice on the PC and audit.

### 5.10 Connectors
- Catalog of MCP servers (Gmail, Calendar, Drive, GitHub, Slack, Notion, Jira, etc.) and Calango's own MCP.
- Connect once per account; each agent receives only the scopes it needs (none, read, read and write).
- The node also exposes an MCP server, so Claude Code or Codex can talk to Calango.

### 5.11 Skills
- Reusable recipes in Markdown.
- When the agent repeats a workflow, it proposes becoming a skill; the person approves it.

### 5.12 Memory
- Facts, preferences and decisions, per account, per agent and per project.
- Everything is reviewable, editable and deletable. The agent proposes changes to its own instructions; the person accepts or refuses them.

### 5.13 Files and documents (later phase)
- A deliverables folder for agents, with a list of who can see each folder.
- Preview of PDF, image, video, spreadsheet and text, with a copy button.

### 5.14 Vault
- Secrets encrypted by the operating system; the agent uses them without seeing the value.
- Importing passwords exported from the browser in a later phase.

### 5.15 Usage and costs
- Panel per agent, routine, project and node: tokens, cost, time.
- Limits per agent and an alert before exceeding them.

### 5.16 External channels (later phase)
- Telegram for approvals and conversation.
- WhatsApp, Slack and email as input and output for the agents, with sending limits and approval.

### 5.17 Teams (Team plan)
- Members, roles, per-channel permissions, shared agents, delegated approval.

---

## 6. Security and privacy

| Threat | Defense |
|---|---|
| Unauthorized access to the node | single-use code pairing, per-device token, revocation, Tailscale |
| Agent does something destructive | mandatory approval for sensitive actions, isolation, restricted working folder |
| Prompt injection via web, email or file | external content marked as data; write tools always require approval |
| Key leakage | OS vault, secrets never go to the client, masking in logs |
| Uncontrolled spending | per-agent budget checked before every call |
| Remote control abused | opt-in on the PC, visible notice, audit, ends with one click |
| Data exfiltration | list of allowed destinations per agent (later phase) |

Audit: every remote action and every approval is recorded and visible on the PC and on the phone.

---

## 7. Data and synchronization

- Each node has its own SQLite.
- **Event log** (append-only) per node, replicated between the account's nodes. Each record has an owner node; there is no concurrent editing of the same record.
- **Code** synchronizes only through git.
- **Large files** stay on the node where they were created; the other node fetches them on demand.
- **Backup**: a documented command for the Docker volume and for the PC's data folder.
- When a node goes down, clients show which one, since when, and what is waiting for it (a lesson from agentes.chat, where only "waiting for computer" appears).

---

## 8. Monetization (summary)

| | Free | Pro | Team |
|---|---|---|---|
| PC node | 1 | 1 | per member |
| VPS node | — | 1 | per member |
| Phone (via Tailscale) | yes | yes | yes |
| Hybrid execution and 24/7 routines | — | yes | yes |
| Phone push | limited | yes | yes |
| Remote screen control | — | to be decided | yes |
| Channels with people | — | — | yes |

Paid extras: model credits (the free plan uses its own key or subscription), VPS managed by us, storage, premium connectors. The VPS node license is signed and validated offline; without a license, it downgrades to free mode instead of stopping. Details in [MONETIZATION.md](MONETIZATION.md) and [PRODUCT_HYBRID.md](PRODUCT_HYBRID.md).

---

## 9. Distribution

| Platform | Format | Caveat |
|---|---|---|
| Windows | signed NSIS, auto-update | code-signing certificate, Git for Windows for the ADE |
| macOS | signed and notarized DMG | Accessibility and Screen Recording permissions for computer use |
| Linux | `.deb`, AppImage | unlocked keyring for the vault |
| VPS | Docker image + one-line installer | data volume, backup, Tailscale |
| iOS | App Store (TestFlight first) | purchase through Apple's system |
| Android | Google Play | purchase through Google's system |

Languages from the start: Portuguese (Brazil), English and Spanish, with a versioned translation catalog.

---

## 10. Monorepo

```
calango/
├── cmd/
│   ├── calango/           core (PC and VPS)
│   └── calango-cloud/     our services
├── internal/              Go: core, api, store, gateway, tools, browser, ade, sync
├── api/                   openapi.yaml (source of both SDKs)
├── apps/
│   ├── desktop/           Electron (main + React renderer)
│   ├── mobile/            Flutter (iOS, Android)
│   └── web/               web build of the renderer
├── packages/
│   ├── sdk-ts/            TypeScript client generated from OpenAPI
│   ├── sdk-dart/          Dart client generated from OpenAPI
│   ├── tokens/            design tokens (JSON → CSS and ThemeData)
│   ├── ui/                React components for desktop/web
│   └── i18n/              single catalog (→ JSON and ARB)
├── deploy/                Dockerfile, compose, installer
└── DEV/                   living documentation
```

---

## 11. Roadmap

| Phase | Delivery | Done criterion |
|---|---|---|
| **0. Foundation** | Go skeleton, docs, Docker (done, not compiled at the time) | `go test` and `/healthz` passing |
| **1. Local desktop MVP** | Electron + `calango`; 1 agent via harness; chat; memory; routines; approvals; usage panel | install on Windows, talk, create a routine, approve an action |
| **2. Phone** | Flutter app; pairing; Tailscale; push via relay | approve a PC action from the phone away from home |
| **3. VPS and first revenue** | Docker image; license; hybrid execution; billing | a routine moves to the VPS with the PC off; Pro subscription charged |
| **4. ADE** | projects, worktrees, editor, terminal, diff, PR | an agent opens a PR from a task, on the PC or the VPS |
| **5. Browser and computer use** | internal browser with take-over; app use; screen control | an agent fills in a form and the person takes over mid-way |
| **6. Channels and teams** | channels with people, router, Team plan | two members and three agents in one channel |
| **7. Expansion** | own relay, external channels, voice, files, marketplace | — |

---

## 12. Competitive implications

- The updated comparison and its official sources are in [COMPETITIVE_LANDSCAPE.md](COMPETITIVE_LANDSCAPE.md); its conclusions and limits should not be replaced by historical notes from an older scan.
- Runs must survive reconnection and clearly indicate failure, interruption and known/estimated/unknown cost.
- Approval must be immutably bound to the exact action; the execution receipt is separate from the decision.
- Routines use an identifier per occurrence, recovery of missed times and idempotent retries; exactly-once execution of external effects is not promised without cooperation from the destination.
- The ADE delivers reviewable evidence (diff, commands and test results); merge and publication still require human approval.
- Harness errors, node disconnection, cost and browser usage must be states the user can understand, not empty responses.

---

## 13. Metrics

Activation (first routine created within 24 h), weekly retention, free→Pro conversion, active VPS nodes, model cost per active user, approvals answered from the phone, routines run successfully.

## 14. Open decisions

1. ~~Mobile~~: decided, Flutter (ADR 0004).
2. ~~Code license~~: decided, AGPL-3.0 with commercial license, CLA and trademark policy (ADR 0010); legal review before publication is still pending.
3. **Screen control**: Pro or only Team (D2).
4. **First MVP harness**: Claude Code, Codex or direct API (D3); blocks the implementation that depends on the harness, not the SQLite store.
5. **Go module name and domain** (D4).
6. **tsnet**: Tailscale license and commercial limits before embedding (D5).
7. **Automatic Maestro update**: keep off by default until the default is confirmed (D6).

## 15. Risks

| Risk | Mitigation |
|---|---|
| Scope too large | phases with done criteria; do not start the next phase early |
| Model cost on the free plan | the free plan uses its own key or subscription |
| Dependence on third-party CLIs | adapter layer; contract tests per harness |
| Dependence on Tailscale | own relay on the roadmap (phase 7) |
| Stores rejecting remote control | review the guidelines before phase 2 |
| Security of computer use | off by default, approval and audit |
