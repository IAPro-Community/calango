# Module: memory

Phase 1 · Package `internal/memory` · Tables `memories`, `skills`

Source: ADR 0011 (P2). Ported from `backend/app/repositories/memory.py`, `backend/app/agents/memory_scope.py` and `backend/app/schemas/memory.py` in `vstorm-co/agenticos` at a4b0b3c.

## Goal
The agent remembers facts, preferences and decisions, and the person controls everything.

## Rules
- Scopes in V1: account and agent. Project scope is deferred until the projects table exists.
- The agent saves memory through a tool (`memory_save`); the person sees, edits and deletes it.
- Context injection: memories from the scope, most recent and most relevant first, up to a fixed token budget.
- Deletion is logical (`deleted_at`) and disappears from the context immediately; physical cleanup after 30 days.
- The agent can **propose** a change to its own instructions; it becomes a card the person accepts or refuses.
- Skills: the agent proposes becoming a skill when it repeats a workflow; the person approves it before it takes effect.

## API
`GET/POST /v1/memories`, `PATCH/DELETE /v1/memories/{id}`, `GET/POST /v1/skills`, `PATCH /v1/skills/{id}`.

`/v1/skills` stays reserved for approved skills (issue #124). Skills imported from Claude Code and Codex use `/v1/skill-imports` and a file index, not the `skills` table (`SPECS/modules/skills.md`, ADR 0022); an import can later be promoted into `skills` with `source='import'`.

## Acceptance
- A deleted memory does not appear in the next run.
- A refused instruction proposal does not change the agent.

## Current state (F1-T11, partial)
- Done: domain `core.Memory` (scopes account/agent, kinds fact/preference/decision, 2000-character limit), migration `0006_memories.sql`, store CRUD and `PurgeMemories`, service `internal/memory` (create, list, update, logical delete, purge, `Context` under a 1500-token budget with the most recent first), injection into runs through `conversations.Config.Memory`. The prompt is unchanged when there are no memories. Tests: `internal/memory/service_test.go` and `TestRunPromptCarriesOnlyActiveMemories`.
- Served: HTTP `/v1/memories` (list, create, patch, delete; F1-T13); `memory_save` is agent-scoped and runs through MCP (ADR 0018). `RunPurge` physically removes expired logical deletions at startup and daily; wired in `cmd/calango/main.go` and tested. Pending: owner addendum for channel/agent action memory; instruction proposals and skills are not in the current V1 implementation.
- Relevance: V1 ranks by recency only; semantic relevance is out of scope for V1.
- Deleting an agent deletes its agent-scoped memories (foreign key cascade).

## Adendo: channel and agent actions as memory base (owner, 2026-11-23)

- The intelligence layer is the Orquestrador Maestro (ADR 0009). Its rules do not alter memory (`DEV/ORQUESTRADOR_MAESTRO.md`).
- Besides explicit memories, actions in channels and agents are saved and used as the base of the memory context.
- Open, before code: which actions are captured (run outcomes, approvals, channel messages, tool calls); what text is stored (no secrets, no raw tool output); whether they are rows in `memories` with a source or a separate table; retention and per-item deletion; their share of `ContextBudget`.
