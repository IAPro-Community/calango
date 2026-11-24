# ADR 0005 — Tailscale as the initial transport

Status: accepted · 2026-11-23

## Context
The phone must reach the PC (behind NAT) and the VPS from anywhere, with end-to-end encryption, without the person configuring a router.

## Decision
Phases 2–6 use Tailscale (the person's own tailnet). Evaluate `tsnet` (a Go library) to embed in `calango` and drop the separate client on the PC and the VPS. An own relay is for phase 7.

## Open items before implementing
- License is BSD 3-Clause; unresolved: whether embedding `tsnet` in a commercial, customer-facing product is permitted under the Tailscale Terms. Obtain written vendor/legal confirmation or choose another transport.
- Personal plan limits are documented below; owner must decide whether to accept them for the intended use. See the research section.

## Research (2026-11-23)

- Tailscale describes `tsnet` as a Go library for embedding Tailscale in a program. The repository's root license is BSD 3-Clause: [tsnet documentation](https://tailscale.com/docs/features/tsnet), [license](https://github.com/tailscale/tailscale/blob/main/LICENSE).
- The official Personal plan is $0 and lists unlimited user devices, up to 6 users, 3 ACL groups, 50 tagged resources and 1,000 ephemeral-resource minutes/month; Standard is listed at $8/user/month: [pricing](https://tailscale.com/pricing), accessed 2026-11-23.
- Terms of Service (updated 2026-08-25) allow use for personal or internal business purposes according to plan (§2.1) and restrict commercial exploitation/resale of Services (§2.3). They do not expressly answer embedding `tsnet` in a commercial, customer-facing product. This is not a legal determination: obtain written Tailscale/legal clearance or choose another transport before implementing F2-T02. [Terms](https://tailscale.com/legal/terms).

## Consequences
- No relay cost for us in this phase; monetization comes from the VPS license, credits and managed VPS.
- Third-party dependency: the transport sits behind an interface (`internal/transport`) so it can be swapped for an own relay.
