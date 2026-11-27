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

## Shared-consumer migration follow-up

Nexus PR IAPro-Community/nexus#243 removes its independent Codex RPC and quota parser, version cache, identity-hint algorithm and idle freshness calculation. Calango now pins the identical SDK candidate `v0.0.0-20261127234402-2b78728e4d48` (consumer-pin job 140789624649) and calls the SDK observation freshness contract for both the read model and projection. Calango's dedicated quota adapter already uses the same SDK RPC, parser and credential-scope implementation. The Nexus-specific display/duplicate hint is not proof of identity and must never be used to authorize Calango account switching or removal. Calango has no repeated CLI-version endpoint, so it does not create unnecessary version subprocesses merely to consume the SDK version cache.

The remaining Nexus-specific quota cache, additional provider identity adapters and interprocess coordination are not certified as migrated. The original Nexus ephemeral credential-home policy is not introduced into Calango. Both consumers require exact-SHA CI and independent review before merge.

The two consumer PRs now pin `v0.0.0-20261127234706-ce1d3a7b9ea2` (SDK consumer-pin job 140790495784). Nexus additionally delegates the full app-server usage cache and Cursor identity hint parsing; Calango benefits from the shared cache and now deduplicates fallback candidates by SDK physical key across profile aliases. Cursor support remains a separate provider-adapter task in Calango; no unverified hint is used for dispatch or profile deletion.

### Provider management follow-up
SDK #12 now owns typed host bindings, bounded discovery, installation inventory state, portable account key/label contracts, and canonical ZCode/AGY/OpenCode/Cursor specifications. Nexus #243 removes its independent registry and installation state rules. Calango uses SDK AccountManager to resolve authorized host gateway factories instead of a hardcoded dispatch switch, with the existing Codex and Anthropic allowlist intact. This does not automatically enable unsupported providers, import credentials, or move persistence and authorization into SDK. Both consumers pin `v0.0.0-20261127235303-beb56ec21712` (consumer-pin 140792166547).
