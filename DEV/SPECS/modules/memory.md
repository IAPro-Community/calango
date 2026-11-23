# Module: memory

Phase 1 · Package `internal/memory` · Tables `memories`, `skills`

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
