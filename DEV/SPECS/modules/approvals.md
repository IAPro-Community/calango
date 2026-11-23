# Module: approvals

Phase 1 · Package `internal/approvals` · Tables `approvals`, `approval_rules`, `audit_log`

## Goal
No sensitive action happens without a person approving it.

## Rules
- Sensitive actions (fixed list in the code): send message/email, delete, publish, pay, deploy, merge, shell command outside the safe list, write through a connector with `write` access.
- The run pauses in `waiting_approval` until the decision. It expires after 24 h (configurable) → `expired`; the run continues knowing it was not approved.
- The card shows the **exact action, target and arguments**.
- The approval points to an immutable snapshot of the action. A SHA-256 digest of the canonical payload is stored with the approval and checked immediately before execution; any difference invalidates the decision and requires a new card.
- The executor uses the approved snapshot, never arguments recomputed by the model after the decision.
- Receipt afterwards: `completed`, `failed` (with reason) or `not_run`.
- "Always approve" stores a rule per agent and action type.
- Approvals from the same run are presented in the order they were requested (`seq`).
- The agent never decides an approval. A decision requires a paired device of a person.
- Every decision goes to `audit_log`.

## API
`GET /v1/approvals?status=pending`, `POST /v1/approvals/{id}/decide` `{ decision: "approve"|"deny", remember: bool }`.

## Events
`approval.requested`, `approval.decided`.

## Acceptance
- An approved action that fails shows `failed` with a reason, not "approved".
- Changing the target or arguments after approval blocks execution and creates a new request.
- With an "always approve" rule, the same action does not generate a card, but does generate an audit entry.

## Status (F1-T08, partial)
- Done: canonical action and SHA-256 digest (`internal/core/approval.go`); migration `0004_approvals.sql` (snapshot columns, status and outcome change once, `approval_rules`); store (`internal/store/approvals.go`); service `internal/approvals` (request, decide, execute, sweep, "always allow" rules, 24 h expiry enforced in `Decide` and `Sweep`).
- HTTP list and decide are served (F1-T13). Pending: the card (client side, F1-T20+); run state `waiting_approval` wired to the executor; reconciliation of an approved action that crashed before its receipt (it stays without outcome); `Request` does not check that `runID` belongs to the agent.
