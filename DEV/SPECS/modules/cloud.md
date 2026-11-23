# Module: cloud (Calango Cloud)

Phases 2–3 · `cmd/calango-cloud` (Go) · Postgres

## Goal
Our own services, kept minimal: account, devices, license, billing, push, updates. **Never** conversations, files or model keys.

## Services
| Service | Routes | Phase |
|---|---|---|
| account | `/v1/auth/*`, `/v1/me`, `/v1/nodes` | 2 |
| push | `/v1/push/register`, `/v1/push/send` | 2 |
| license | `/v1/license/refresh` | 3 |
| billing | `/v1/billing/checkout`, Stripe/Apple/Google webhooks | 3 |
| updates | `/v1/updates/{channel}/{platform}` | 1 |

## Rules
- Login by email with a magic link and OAuth (Google, Apple, GitHub).
- Telemetry only with opt-in, without content.
- One binary, services as modules; deployed in a container.
- Postgres database; immutable migrations.

## Acceptance
- Data audit: no Cloud table has a column holding conversation text.
