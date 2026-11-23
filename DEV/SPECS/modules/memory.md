# Module: memory

Phase 1 · Package `internal/memory` · Tables `memories`, `skills`

Source: ADR 0011 (P2). Ported from `backend/app/repositories/memory.py`, `backend/app/agents/memory_scope.py` and `backend/app/schemas/memory.py` in `vstorm-co/agenticos` at a4b0b3c.

## Goal
The agent remembers facts, preferences and decisions, and the person controls everything.

## Rules
- Scopes: account, agent, project.
- The agent saves memory through a tool (`memory_save`); the person sees, edits and deletes it.
- Context injection: memories from the scope, most recent and most relevant first, up to a fixed token budget.
- Deletion is logical (`deleted_at`) and disappears from the context immediately; physical cleanup after 30 days.
- The agent can **propose** a change to its own instructions; it becomes a card the person accepts or refuses.
- Skills: the agent proposes becoming a skill when it repeats a workflow; the person approves it before it takes effect.

## API
`GET/POST /v1/memories`, `PATCH/DELETE /v1/memories/{id}`, `GET/POST /v1/skills`, `PATCH /v1/skills/{id}`.

## Acceptance
- A deleted memory does not appear in the next run.
- A refused instruction proposal does not change the agent.

## Current state (F1-T11, partial)
- Done: domain `core.Memory` (scopes account/agent, kinds fact/preference/decision, 2000-character limit), migration `0006_memories.sql`, store CRUD and `PurgeMemories`, service `internal/memory` (create, list, update, logical delete, purge, `Context` under a 1500-token budget with the most recent first), injection into runs through `conversations.Config.Memory`. The prompt is unchanged when there are no memories. Tests: `internal/memory/service_test.go` and `TestRunPromptCarriesOnlyActiveMemories`.
- Served: HTTP `/v1/memories` (list, create, patch, delete; F1-T13). Pending: `memory_save` tool (needs the first harness, D3); scope `project` (no `projects` table yet); purge schedule (method exists, no loop); instruction proposals and skills as cards (F1-T13 or later).
- Relevance: V1 ranks by recency only; semantic relevance is out of scope for V1.
- Deleting an agent deletes its agent-scoped memories (foreign key cascade).
