# ADR 0008 — Event-based replication between nodes; code through git

Status: accepted · 2026-11-23

## Context
PC and VPS work together (hybrid execution). Conversations, memory, routines and tasks must appear on both. Project code too.

## Decision
- Each node writes an append-only log (`events`) with a unique `(node_id, seq)`.
- Nodes exchange events through the transport; each one applies the other's events and stores `last_applied_seq` in `peers`.
- Each entity has an **owner node**; only the owner changes it. The other node requests a change through a command, which the owner applies and publishes. There is no concurrent editing, so there is no merge.
- Code is synchronized only through git (push/pull between worktrees and the remote). No custom file sync.
- Large files stay on the origin node; the other node fetches them on demand.

## Consequences
- Simple to reason about; no CRDTs.
- If the owner is offline, the change waits (the interface shows this).
- A routine with `run_on=auto` changes owner when the PC goes offline: the ownership transfer is an explicit event.
