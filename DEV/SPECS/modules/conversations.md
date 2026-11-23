# Module: conversations

Phase 1 · Package `internal/conversations` · Tables `conversations`, `conversation_members`, `messages`, `runs`, `run_steps`

## Goal
DMs with agents and, later, channels. Each message can trigger a run; the run publishes steps in real time.

## Rules
- One run per message per agent. Several runs in parallel in the same conversation up to a configurable limit (default 3); beyond that, a queue.
- Repeated requests caused by timeout/reconnection carry a client idempotency key; the same message does not create duplicate runs.
- Per-agent option: "one reply at a time".
- **Stop** ends only that run. **Guide** injects an instruction into the next step of the run in progress, without becoming a message.
- **Retry with instructions**: redoes a stopped run with an extra direction, in the same place.
- Steps (`run_steps`) are recorded and streamed: thinking (optional), tool call, result, text.
- The WebSocket connection is transport, not the source of truth: after reconnecting, the client fetches messages, run state and the persisted step sequence.
- When the process crashes, runs without a recoverable checkpoint end as `failed/interrupted`; only a harness with an explicit checkpoint may resume without starting another execution.
- Next steps suggested by the agent become a card of checkboxes, nothing checked by default; it does not block.
- Requests between agents appear folded, outside the unread count.

## API
`GET/POST /v1/conversations`, `GET /v1/conversations/{id}/messages`, `POST /v1/conversations/{id}/messages`, `POST /v1/runs/{id}/stop`, `POST /v1/runs/{id}/steer`, `POST /v1/runs/{id}/retry`.

## Events
`message.created`, `run.started`, `run.step`, `run.finished`.

## Acceptance
- Two consecutive messages create two parallel runs; stopping one does not affect the other.
- A run interrupted by a `calango` crash appears as `failed` with `error_code=interrupted` when it comes back.
- Repeating the same message request returns the same created resource.

## Status (F1-T07, partial)
Implemented in `internal/conversations` (service) and `internal/store/conversations.go` (persistence, migration 0003):
- DM creation; message and run admitted in one transaction; repeated idempotency key returns the stored message and run (`runs.idempotency_key`).
- Parallel limit per conversation (default 3, `Config.ParallelLimit`); runs beyond it stay `queued` and start when a slot frees.
- Stop: a running run is cancelled and ends `stopped`; a queued run ends `stopped` at once. Other runs are untouched.
- Retry: only `stopped` or `failed` runs; a new run in the same conversation, same source message, `retry_of` set, instructions appended to the prompt.
- Paused node: the message is stored and the run is recorded `failed/agent_paused`; the model is not called.
- Startup (`Service.Start`): runs left `running` end `failed/interrupted`; queued runs are dispatched again.
- Steps: text chunks recorded in `run_steps` with `seq`, written even if the run is cancelled.
Not yet: guide (steer), per-agent "one reply at a time", WebSocket reconnection (F1-T13), HTTP routes, routine/approval triggers. Run cost stays `unknown` because the gateway does not return cost to the caller.
