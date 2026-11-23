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
