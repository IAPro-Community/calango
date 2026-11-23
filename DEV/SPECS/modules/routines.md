# Module: routines

Phase 1 (cron/interval) · Phase 3 (events, hybrid) · Package `internal/core/scheduler.go` + `internal/routines` · Tables `routines`, `routine_runs`

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
- A routine can end itself (`routine_finish` tool).
- Routines do not appear on the task board.

## Current state
`core.Scheduler` with an interval and an injectable clock already exists and has tests (not run).

## API
`GET/POST /v1/routines`, `PATCH/DELETE /v1/routines/{id}`, `POST /v1/routines/{id}/run`, `GET /v1/routines/{id}/runs`.

## Events
`routine.fired`, `routine.failed`.

## Acceptance
- A transient failure followed by success on retry: a `routine_run` with `attempt=2`, without a failure notification.
- PC off for 3 scheduled times: when it comes back, only one execution.
- Redelivering the same occurrence after a restart recovers the same `routine_run` record.
