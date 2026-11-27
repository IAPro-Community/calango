# Capacity and account admission acceptance (#201)

Status: experimental; not approved for production. The host owns authorization, durable accounting and process isolation. IAPro-SDK owns verified identity, quota normalization, fallback eligibility and managed lease coordination. Never copy provider credentials or use account switching to bypass provider limits.

## Reconciliation

A timeout, cancellation or ambiguous failure is not proof that a provider's descendants stopped. Ledger completion is necessary but not sufficient. The reservation remains until independent reconciliation. Ordinary successful turns require the SDK terminal event and direct process exit; this does not certify process-tree containment. Until #113 proves containment, do not enable automatic routing in production.

1. Stop admission of new work for the affected node/account. Stop the scheduler, daemon and any other authorized consumers of that physical account; do not disable the routing flag as a way to bypass a fence.
2. Identify the affected owner/provider, hashed physical account key, holder, fence and expiry using a read-only inspection of the stopped node's database. Preserve a consistent backup and ledger/audit records; never include tokens or raw provider IDs in evidence.
3. Independently verify quiescence at the OS/container boundary (including descendants and other consumers), resolve any ambiguous external actions and reconcile missing accounting. A PID disappearing, TTL expiring, or restarting the daemon is not enough.
4. The shared coordinator's ReconcileExpired requires the exact expired fence and an independently established true quiescence proof. No supported operator command exists yet; do not edit or delete SQLite lease rows manually. Keep the profile disabled/isolated and escalate to maintainers until a reviewed offline reconciliation operation is available.
5. After a supported reconciliation, require a fresh identity-bound snapshot and check the new fence generation. Never reuse pre-failure quota evidence or replay an ambiguous turn.

## Real acceptance matrix (not performed by CI)

- Use an isolated dedicated Codex profile with operator-approved terms, explicit budget and identity-bound zero-marginal-cost tariff; no credential copying. Record the exact Calango and SDK SHAs, CLI and app-server versions, OS, sanitized statuses and test evidence.
- Confirm a read-only quota probe, scope, exact model group, reset, freshness, rate limit, expired login and unsupported provider. Never substitute a fabricated snapshot for live evidence.
- Demonstrate eligible primary, verified exhausted primary and authorized fallback before dispatch; stale/unknown/paid/incompatible cases must fail closed. Do not deliberately exhaust a subscription or evade limits; use fixtures for synthetic edge cases.
- Run isolated contention between probe and turn, alias contention, cancel/timeout, renewal loss, ledger failure, restart and rollback. Prove process-tree quiescence independently. No cross-account session continuation without verified provider support.
- Keep automatic routing disabled until the independent review and real acceptance gates of #201/#113 pass. Nexus parity must be verified against the actual release branch, not its default branch alone.

## Nexus release parity (2026-11-27)

Audited Nexus release/v1.0.0 at 0ceff448b1d5ac10e0268fc04fc7656b94deff7c, not only its default branch. Nexus pins SDK 8e5bf0f97752; Calango previously pinned 2ba7f61aa3ad. The new Calango candidate consumes SDK v0.0.0-20261127233855-7e4d4249e000 (consumer-pin job 140788194194, SDK PR #12); the actual SDK shared error classifier now projects authentication-required into the host CapacityView and UI. The SDK PR must be reviewed and integrated before the Calango PR is merged.

| Nexus commit | Shared/host treatment |
|---|---|
| 53956e848f36 | Shared structured authentication-expired handling via SDK PR #12; do not copy credentials to temporary homes to bypass interactive locks. SDK batch coordinator already exists; Calango keeps explicit owner and serialized dedicated probes. |
| ed891a0d9b | SDK PhysicalKey already deduplicates lease identity across profile aliases; verified identity parsing and duplicate detection for additional providers need SDK work. Do not automatically remove profiles, bindings or credentials in Calango. |
| 422bd1353f | Calango already skips unsupported providers and caches identity-bound observations with TTL/backoff; Nexus idle/active scheduling is host-specific. |
| 3c21f0aace | Bounded/coalesced CLI version and host-auth probes need a common SDK implementation with identity/environment scope and invalidation. |
| 3e5b75de11 | Shared noncredential build caches are not quota logic; require separate security and tenant-boundary review before any opt-in. |

This is partial functional parity, not proof of full Nexus migration, real provider acceptance or cross-product locking.
