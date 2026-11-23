# Module: agents

Phase 1 · Package `internal/core` (domain) + `internal/agents` (service) · Tables `agents`, `agent_tools`

## Goal
Create and configure agents: role, instructions, harness/model, tools, budget, where they run, approval level.

## Rules
- `token_limit_month > 0` is required. Without a limit, there is no call (already implemented in `core.Agent.Charge`).
- The sum of usage must never overflow an integer or exceed the limit; a refused charge does not change usage.
- `run_on`: `pc`, `vps` or `auto`. `vps` requires a plan with a VPS node.
- `approval_level`: `always` (everything that writes asks), `sensitive` (default), `trusted` (only what the fixed list of dangerous actions defines).
- Global pause: a switch that prevents any new run and stops the ones in progress.
- Reasoning effort uses the same four names everywhere: `low`, `medium`, `high`, `max`.
- Assisted creation: the person describes the agent, an internal agent generates a draft that the person reviews before saving.

## API
`GET/POST /v1/agents`, `GET/PATCH/DELETE /v1/agents/{id}`, `POST /v1/agents/pause-all`, `POST /v1/agents/resume-all`.

## Events
`agent.updated`, `agents.paused`.

## Acceptance
- Creating an agent without a limit fails with `invalid_input`.
- A paused agent does not start a run; a message sent to it is recorded with a warning.
- A change to instructions applies to the next run, not to the one in progress.
