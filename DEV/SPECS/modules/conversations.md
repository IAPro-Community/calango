# Module: conversations

Phase 1 · Package `internal/conversations` · Tables `conversations`, `conversation_members`, `messages`, `runs`, `run_steps`

## Goal
DMs with agents and, later, channels. Each message can trigger a run; the run publishes steps in real time.

## Rules
- One run per message per agent. Several runs in parallel in the same conversation up to a configurable limit (default 3); beyond that, a queue.
- Repeated requests caused by timeout/reconnection carry a client idempotency key; the same message does not create duplicate runs.
- Per-agent option: "one reply at a time".
- **Stop** ends only that run. **Steer** interrupts a queued or running run and queues a continuation that carries the owner's guidance; the interrupted run ends `stopped` and keeps its partial text. Guidance does not become a message.
- **Retry with instructions**: redoes a stopped run with an extra direction, in the same place.
- Steps (`run_steps`) are recorded and streamed: thinking (optional), tool call, result, text.
- The WebSocket connection is transport, not the source of truth: after reconnecting, the client fetches messages, run state and the persisted step sequence.
- When the process crashes, runs without a recoverable checkpoint end as `failed/interrupted`; only a harness with an explicit checkpoint may resume without starting another execution.
- Next steps suggested by the agent become a card of checkboxes, nothing checked by default; it does not block.
- Requests between agents appear folded, outside the unread count.

## API
`GET/POST /v1/conversations`, `GET/POST /v1/conversations/{id}/messages`, `GET /v1/runs/{id}`, `POST /v1/runs/{id}/stop`, `POST /v1/runs/{id}/retry`, `POST /v1/runs/{id}/steer` (body `{"guidance": "..."}`, `Idempotency-Key` header, 202 with the continuation run).

## Events
`message.created` (data: Message), `run.started` and `run.finished` (data: Run), `run.step` (data: Step with `run_id`). Served by `GET /v1/events` (WebSocket, Q23): owner-filtered, in-memory, no replay. A client 64 events behind is closed with 1013 and reloads over REST (ADR 0015).

## Acceptance
- Two consecutive messages create two parallel runs; stopping one does not affect the other.
- A run interrupted by a `calango` crash appears as `failed` with `error_code=interrupted` when it comes back.
- Repeating the same message request returns the same created resource.

## Status (F1-T07, complete)
Implemented in `internal/conversations` (service) and `internal/store/conversations.go` (persistence, migration 0003):
- DM creation; message and run admitted in one transaction; repeated idempotency key returns the stored message and run (`runs.idempotency_key`).
- Parallel limit per conversation (default 3, `Config.ParallelLimit`); runs beyond it stay `queued` and start when a slot frees.
- Stop: a running run is cancelled and ends `stopped`; a queued run ends `stopped` at once. Other runs are untouched.
- Retry: only `stopped` or `failed` runs; a new run in the same conversation, same source message, `retry_of` set, instructions appended to the prompt.
- Steer (`Service.Steer`): a queued run stops at once; a running run is cancelled and its partial text stays in its steps. Either way a continuation is admitted in the same conversation and source message, with `steer_of` set and `instructions` = guidance. Its prompt holds the reply so far and the guidance (`steeredPrompt`). If the run finishes before the cancel lands, it stays `done` and the continuation still runs. The same idempotency key returns the same continuation. Different guidance while one is pending, or Stop on a run with a pending steer: `409`. Steering a run that is not queued or running: `409`. Empty guidance: `400`. A crash while a steer is pending loses the continuation (the run ends `failed/interrupted`). If admitting the continuation fails after the run stopped, the error is logged and the guidance is lost.
- Paused node: the message is stored and the run is recorded `failed/agent_paused`; the model is not called.
- Startup (`Service.Start`): runs left `running` end `failed/interrupted`; queued runs are dispatched again.
- Steps: text chunks recorded in `run_steps` with `seq`, written even if the run is cancelled.
- On successful message-triggered runs, the text steps are joined and persisted as an `AuthorAgent` message in the same transaction as run completion; `message.created` publishes only after commit. The conversation history is durable across reloads.
- Provider chain (ADR 0016): a run tries the agent's `providers` in order. The next entry runs only when the attempt failed before its first output chunk: `ErrUnavailable`, `harness.ErrNotInstalled` or `harness.ErrSignedOut`. Budget, cancellation and ledger errors never fall back. Each attempt is one gateway call and one `provider_attempt` step (`account`, `model`, `outcome`, `error_code`). An agent with no entries fails with `no_provider`.
- Per-agent single-active-run FIFO, persisted assistant replies, idempotent resends and REST recovery satisfy F1-T07. WebSocket event replay is intentionally not part of the event contract (ADR 0015); the client reloads messages, run state and persisted steps over REST. Codex cost remains unknown without configured pricing.
