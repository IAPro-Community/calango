# AGENTS.md — Calango

Entry point for any coding agent (Claude Code, Codex, Cursor, etc.) working in this repository.

## What it is

Calango is a personal agent that runs on the person's computer (PC node, free), extends to their VPS (VPS node, paid), and is controlled from anywhere by phone (iOS/Android) and desktop. Core in Go, desktop in Electron, mobile in Flutter.

## Reading order (required before coding)

1. `GOVERNANCE.md` and `CONTRIBUTING.md`.
2. Assigned GitHub issue, acceptance criteria, dependencies and linked PRs.
3. `DEV/INDEX.md` and relevant architecture/ADRs.
4. Module specification under `DEV/SPECS/modules/` and relevant code/tests.

Do not load the whole DEV directory or assume a global ACTIVE/HANDOFF file defines your assignment. Use one isolated worktree per concurrent session and an issue-linked branch from main. Confirm the issue is Ready and a human is accountable. Do not take another contributor's issue without coordination.

## Non-negotiable rules

- **Original code.** agentes.chat (`agentes-chat/agentes-chat`) was read only as a product reference. Do not copy code, text, tests or assets from it. agentes.chat is AGPL; any copied fragment would contaminate the license.
- **Exception: Orquestrador Maestro** (`IAPro-Community/Orquestrador-Maestro`, Apache-2.0). Do not copy the repository: consume it from the official npm package (ADR 0009). Keep `NOTICE` up to date.
- **Exception: AgenticOS** (`vstorm-co/agenticos`, Apache-2.0): only the modules listed in ADR 0011, rewritten in Go, TypeScript and Dart with attribution. Keep `NOTICE` and `third_party/agenticos/` up to date.
- **The binary is called `calango`** (not `calangod`). Entry point in `cmd/calango`.
- **Clients do not run agents.** Desktop, mobile and web only talk to the `calango` API.
- **Every model call goes through the gateway** (`internal/gateway`): budget checked before, cost recorded after.
- **Sensitive actions require approval** (send, delete, publish, pay, deploy, destructive command).
- **Secrets never go to the client** nor to logs.
- **Migrations are immutable** once committed: a change means a new migration.
- **New architecture decision = new ADR** in `DEV/ADR/`.
- Commit/push/open PR only within explicit delegated authority, on the assigned branch. Never push directly to main, self-approve, bypass checks, merge or publish autonomously.
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

Record implementation scope, exact commands and actual test results, limitations, blockers and next steps in the assigned issue/PR. Update specifications or ADRs only when the change requires them. Do not routinely modify shared ACTIVE, HANDOFF, WORKLOG, VERIFY or roadmap status files; preserve them during the backlog transition. Request independent human review and leave the issue open until acceptance and any required post-merge validation.
