# Module: agents

Phase 1 · Package `internal/core` (domain) + `internal/agents` (service) · Tables `agents`, `agent_tools`

## Goal
Create and configure agents: role, instructions, provider chain (ordered `{account, model}` entries, ADR 0016), tools, budget, where they run, approval level.

## Rules
- `token_limit_month > 0` is required. Without a limit, there is no call. Enforced by `core.Agent.Validate` on save and by `internal/gateway` before each call.
- The sum of usage must never overflow an integer or exceed the limit; a refused charge does not change usage.
- `run_on`: `pc`, `vps` or `auto`. `vps` requires a plan with a VPS node.
- `approval_level`: `always` (everything that writes asks), `sensitive` (default), `trusted` (only what the fixed list of dangerous actions defines).
- Global pause: a switch that prevents any new run and stops the ones in progress.
- Reasoning effort uses the same four names everywhere: `low`, `medium`, `high`, `max`.
- Assisted creation: the person describes the agent, an internal agent generates a draft that the person reviews before saving.

## API
`GET/POST /v1/agents`, `GET/PUT/DELETE /v1/agents/{id}`, `POST /v1/agents/pause-all`, `POST /v1/agents/resume-all`.

## Events
`agent.updated`, `agents.paused`.

## Acceptance
- Creating an agent without a limit fails with `invalid_input`.
- A paused agent does not start a run; a message sent to it is recorded with a warning.
- A change to instructions applies to the next run, not to the one in progress.

## Status (F1-T06)
- Done: create, list, get, replace (PUT, full replacement), delete (409 `conflict` when runs or usage exist), node-wide pause and resume, monthly token usage per agent (UTC month). Domain `core.Agent`, service `internal/agents`, store `internal/store/agents.go`, table `node_state` (migration 0002).
- Decision: `PATCH` replaced by `PUT`. The contract already requires `name` and `token_limit_month`, so a partial update adds nothing.
- Not in F1-T06: pause enforcement on new runs and stopping runs in progress (F1-T07); `agent.updated` and `agents.paused` events (WebSocket, F1-T13); per-agent pause (`agents.paused` column stays 0); `run_on=vps` plan check (F3); `agent_tools` table (not yet in a migration). HTTP handlers were added in F1-T13 behind the bearer token; device pairing is F1-T15.
- Q32 (ADR 0016): `harness` and `model` are replaced by `providers`, an ordered chain of `{account_id, model}` (1–8 entries, required on create). Responses carry `account` (`<kind>/<label>`). Accounts are managed at `/v1/provider-accounts` (`internal/providers`). Migration 0010 moves each agent's old harness and model into the owner's `default` account; agents without a model get no chain and must be saved again.
