# Module: nodes-sync

Phase 3 · Packages `internal/sync`, `internal/transport` · Tables `nodes`, `events`, `peers` · ADR 0008

## Goal
The PC and the VPS of the same account work as a single assistant.

## Rules
- Each domain write produces an event in `events` with an increasing `(node_id, seq)`.
- Exchange: on connecting, each node asks the other for events with `seq > last_applied_seq`; then a continuous stream.
- Applying an event and advancing `last_applied_seq` happen in the same transaction; a repeated `(node_id, seq)` is recognized without reapplying the change.
- Each entity has an owner node. The other node requests a change through a command (`POST /v1/sync/commands`), which the owner applies.
- A command has an idempotent ID, an owner, an expiry and a result; a replicated event does not run an external effect again.
- `run_on=auto`: when the PC has been offline for more than 2 min, the VPS takes over the auto routines (event `ownership.transferred`). When the PC comes back, the routines return to it at the end of the run in progress.
- Heartbeat every 15 s; offline after 45 s without a response. Event `node.offline` with `since`.
- Code: worktrees sync through git; Calango only orchestrates `push`/`pull`.
- Failover offers at-least-once delivery. External effects need an idempotent key on the destination or an approval/receipt that makes duplication visible; do not promise distributed exactly-once.

## Acceptance
- A message created on the PC appears on the VPS in less than 2 s with both online.
- PC off: an auto routine runs on the VPS at its next time; when the PC comes back on, it sees the result.
