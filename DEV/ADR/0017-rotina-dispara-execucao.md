# ADR 0017 — Routine fires an agent run, with a control line for news and finish

Status: accepted · 2026-11-23 · owner decision (Q33) · builds on ADR 0016 and the conversations service.

## Context

F1-T09 and F1-T13 need routines that actually run an agent. Until now `routines.Executor` was an interface with no implementation and no wiring. The owner also asked for two behaviours that the executor must report back:

- `notify_on = news`: notify only when the run found something new;
- `routine_finish`: the agent can end a routine it no longer needs.

The routine must not get a second, hidden model path. It must use the same conversations and gateway path as a person's message.

## Decision

1. **Execution is a DM run.** Each routine owns one DM conversation, created on its first execution and stored in `routines.conversation_id`. Each attempt sends `core.RoutineMessage(prompt)` into that DM, so the run goes through the conversations service, the provider chain (ADR 0016), and the gateway. Routines hold no model client of their own.
2. **Idempotency.** The message key is `<occurrence_id>#<attempt>`. A retry is a new attempt and so a new run; a crash after the send returns the same run instead of sending again.
3. **Control line.** The message ends with a contract asking the agent to finish its reply with one line:
   `ROUTINE: {"had_news": true|false, "finish": true|false}`.
   The last such line in the reply wins. `core.ParseRoutineSignal` reads it.
4. **Defaults that never drop news.** No line, or a malformed one, gives `had_news = true` and `finish = false`. The owner hears about the run and the routine keeps running. A missing `had_news` key also counts as true. Ending a routine needs an explicit `"finish": true`.
5. **`routine_finish` ends the routine.** `finish=true` on a successful run sets `ended_at` (migration 0012). An ended routine is never scheduled again, cannot be resumed or run now, and can be deleted. Finish on a failed attempt is ignored: the run did not complete the routine's job.
6. **Manual run.** `RunNow` adds an occurrence with ID `<routine>@manual:<id>` that runs at the next tick. It respects the quiet window and the node's pause, and a paused routine cannot run now.
7. **Concurrency.** Ticks run one at a time (`runMu`). Executions do not hold the store mutex, so routine edits are not blocked by a long agent run. The outcome is written against a fresh read of the routine. Known limit: occurrences of one node run one after another.
8. **Validation.** The prompt is validated with the contract included (`ValidateMessageBody(RoutineMessage(prompt))`), so a prompt that becomes too long fails at create or update, not on every retry.

## Consequences

- The model output drives two state fields (`had_news`, `finish`). A model that ignores the contract gets the safe defaults above, and the owner sees it in the conversation.
- The contract text is part of the product: changing it changes behaviour. It lives in `internal/core/routine_signal.go` with its tests.
- A crash between creating the DM and saving `conversation_id` leaves an extra empty DM. Accepted: one per crash in a rare window.
- Node placement stays as in F1-T09: `run_on` pc, vps or auto. V1 wiring runs on the PC node only.
