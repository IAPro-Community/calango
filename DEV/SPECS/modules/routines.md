# Module: routines

Phase 1 (cron/interval) · Phase 3 (events, hybrid) · Packages `internal/core` (`routine.go`, `cron.go`) + `internal/routines` · Tables `routines`, `routine_runs`

Source: ADR 0011 (P2). Ported from `backend/app/services/routine_actions.py`, `routine_delivery.py`, `routine_authority.py` and `routine_sources.py` in `vstorm-co/agenticos` at a4b0b3c.

## Goal
Scheduled work, or work triggered by an event, executed by an agent.

## Rules
- Triggers: `cron` (5-field expression, the account's time zone), `interval`, `event` (connector: new email, PR opened, webhook).
- `run_on`: pc, vps or auto. Auto = PC if online, otherwise VPS; without a VPS, it waits for the PC.
- **Retry**: up to `max_retries` with exponential backoff (1 min, 5 min, 30 min). Exhausted: notifies the owner with the error. Never silently loses a delivery.
- Each scheduled time has a deterministic, persisted `occurrence_id`; resending, restarting and retrying do not create a second occurrence for the same time.
- Retry of an external integration is at-least-once unless the destination accepts an idempotent key; approval is still required for sensitive actions.
- **Silence**: the agent declares whether there was news (`had_news`). `notify_on=news` does not notify without news.
- **Quiet window**: runs inside it are postponed to the end of the window, not discarded.
- Missed routine (PC off at the scheduled time): runs once when it comes back, not N times.
- A routine can end itself: the reply's control line sets `finish: true` (ADR 0017). Later occurrences are not scheduled, and pause, run now and edit return 409.
- Routines do not appear on the task board.

## Current state
Status (F1-T09, cron and interval done): domain (`core.Routine`, `ParseCron`, `Routine.NextAfter`, `PostponeOutsideQuietWindow`, `RetryBackoff`, `OccurrenceID`, `ParseRoutineSignal`), migrations `0005_routines.sql` and `0012_routine_conversation.sql`, store, and `internal/routines.Service` (`Create`, `Get`, `List`, `Update`, `SetPaused`, `Delete`, `RunNow`, `Occurrences`, `Tick`, `Start`, `Run`) are implemented and tested in `internal/routines/*_test.go`.
Firing (Q33, ADR 0017): `ChatExecutor` (`internal/routines/executor.go`) sends `core.RoutineMessage(prompt)` into one DM per routine (`routines.conversation_id`), one agent run per attempt, idempotency key `<OccurrenceID>#<Attempt>`. The reply's last `ROUTINE: {"had_news", "finish"}` line sets news notification and `ended_at`; a missing or malformed line means `had_news=true, finish=false`. `RunNow` queues one `@manual:` occurrence for the next tick. The daemon ticks every 30 s on the PC node (`cmd/calango/main.go`); `run_on=vps` routines wait for a VPS node to run the daemon.
Known limits: a crash between creating the DM and saving `conversation_id` creates a second DM; a failed attempt's run stays in the conversation; a run waiting for approval keeps its attempt open until the decision (Q15); each node runs one occurrence at a time.
Pending: event triggers (phase 3); VPS execution (phase 2 nodes); approval card (client side).

## API
`GET/POST /v1/routines`, `GET/PATCH/DELETE /v1/routines/{id}`, `POST /v1/routines/{id}/run` (202, queues one `@manual:` occurrence), `GET /v1/routines/{id}/runs`. Schemas in `api/openapi.yaml`.

## Events
`routine.fired` and `routine.failed` go to `/v1/events` with `type` set to the kind. `data` is the `routines.Event` struct with Go field names (no JSON tags): `Kind`, `OwnerID`, `RoutineID`, `OccurrenceID`, `Attempt`, and `Error` (set on `routine.failed`).

## Acceptance
- A transient failure followed by success on retry: a `routine_run` with `attempt=2`, without a failure notification.
- PC off for 3 scheduled times: when it comes back, only one execution.
- Redelivering the same occurrence after a restart recovers the same `routine_run` record.
- A reply with `finish: true` ends the routine: no later occurrence runs, and pause, run now and edit return 409.
- Run now queues one `@manual:` occurrence that runs once at the next tick.
- Every attempt of one routine reuses the same DM and sends `<OccurrenceID>#<Attempt>` as its idempotency key.
