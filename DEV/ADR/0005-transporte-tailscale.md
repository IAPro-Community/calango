# ADR 0005 — Tailscale as the initial transport

Status: accepted · 2026-11-23

## Context
The phone must reach the PC (behind NAT) and the VPS from anywhere, with end-to-end encryption, without the person configuring a router.

## Decision
Phases 2–6 use Tailscale (the person's own tailnet). Evaluate `tsnet` (a Go library) to embed in `calango` and drop the separate client on the PC and the VPS. An own relay is for phase 7.

## Open items before implementing
- Confirm the license and terms of `tsnet` for embedded use in a commercial product.
- Confirm the current limits of the free plan for personal use.

## Consequences
- No relay cost for us in this phase; monetization comes from the VPS license, credits and managed VPS.
- Third-party dependency: the transport sits behind an interface (`internal/transport`) so it can be swapped for an own relay.
