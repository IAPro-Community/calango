# AGENTS.md — Calango

Entry point for any coding agent (Claude Code, Codex, Cursor, etc.) working in this repository.

## What it is

Calango is a personal agent that runs on the person's computer (PC node, free), extends to their VPS (VPS node, paid), and is controlled from anywhere by phone (iOS/Android) and desktop. Core in Go, desktop in Electron, mobile in Flutter.

## Reading order (required before coding)

1. `DEV/INDEX.md` — map of all documentation.
2. `DEV/HANDOFF.md` — where the work stopped and the next step.
3. `DEV/CONTEXT.md` — current project state.
4. `DEV/SPECS/ACTIVE.md` — contract of the task in progress.
5. `DEV/SPECS/ROADMAP.md` — phases and numbered tasks (e.g., `F1-T03`).
6. Only then, the spec of the module you are going to touch, in `DEV/SPECS/modules/`.

Do not load the whole `DEV/` folder. Read what the task needs.

## Non-negotiable rules

- **Original code.** Tucano (`eduwxyz/tucano`) and agentes.chat (`agentes-chat/agentes-chat`) were read only as product references. Do not copy code, text, tests or assets from them. agentes.chat is AGPL; any copied fragment would contaminate the license.
- **Exception: Orquestrador Maestro** (`IAPro-Community/Orquestrador-Maestro`, Apache-2.0). Do not copy the repository: consume it from the official npm package (ADR 0009). Keep `NOTICE` up to date.
- **The binary is called `calango`** (not `calangod`). Entry point in `cmd/calango`.
- **Clients do not run agents.** Desktop, mobile and web only talk to the `calango` API.
- **Every model call goes through the gateway** (`internal/gateway`): budget checked before, cost recorded after.
- **Sensitive actions require approval** (send, delete, publish, pay, deploy, destructive command).
- **Secrets never go to the client** nor to logs.
- **Migrations are immutable** once committed: a change means a new migration.
- **New architecture decision = new ADR** in `DEV/ADR/`.
- Do not commit or push without an explicit request from the project owner.
- License: AGPL-3.0 (`LICENSE`). Contributions require accepting `CLA.md`. Do not add third-party code without checking its license.
- **Language: English.** Code, comments, commit messages, issues, pull requests and documentation are written in English (see [DEV/CONVENTIONS.md](DEV/CONVENTIONS.md)). Translations of user-facing text go through the catalog in `packages/i18n`.

## Commands

```bash
go vet ./...
go test ./...
go run ./cmd/calango          # listens on 127.0.0.1:7317
curl 127.0.0.1:7317/healthz
```

Desktop, mobile and cloud do not exist yet; their commands will be added here when they are created.

## After working

Update `DEV/WORKLOG.md` (what changed and why), `DEV/VERIFY.md` (what was verified and how) and `DEV/HANDOFF.md` (next step). Mark the task in `DEV/SPECS/ROADMAP.md`.
