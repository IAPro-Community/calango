# Module: approvals

Phase 1 · Package `internal/approvals` · Tables `approvals`, `approval_rules`, `audit_log`

Source: ADR 0011. Approvals (P2) from `backend/app/services/approvals.py` and `backend/app/agents/approval.py`; audit chain (P1) from `backend/app/core/audit.py` and `backend/app/services/audit.py`; `vstorm-co/agenticos` at a4b0b3c.

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
- HTTP list and decision are served. `TestApprovalDecisionAPIExecutesApprovedSnapshot` creates an approval through `Service.Request`, posts the HTTP decision and verifies the exact approved snapshot reaches a fake executor. The in-conversation card and mocked decision dispatch are implemented. Not verified: a live Codex/MCP write or an action against an external connector; no real model call was authorized.
