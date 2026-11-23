# Module: pairing-auth

Phase 1 (local token) · Phase 2 (pairing) · Package `internal/auth` · Tables `devices`, `pairing_codes`

## Goal
Only authorized devices talk to a node.

## Pairing flow
1. On the desktop (or in the VPS log): "Pair device" generates an 8-character code + QR, valid for 5 min, single use. Stored as a hash.
2. The app sends the code to `POST /v1/pair` through the transport.
3. The node returns a device token (random, 256 bits), stored in the phone's keychain. On the node, only the hash is kept.
4. A device list with last access; revoking invalidates it immediately.

## Rules
- Every route requires `Authorization: Bearer <token>`, except `/healthz` and `/v1/pair`.
- `/v1/pair` has an attempt limit (5 per minute per origin).
- Token scopes: `full`, `approve_only` (for a watch, for example), `read`.
- On the first start of the VPS node, an administrator password is required; no default credential.

## Acceptance
- A code used twice fails the second time.
- A revoked token gets `unauthorized` on the next call.

## Status (V1 local)

- Implemented (V1 local): local bearer token (`internal/auth`, file `DataDir/api-token`, mode 0600, refused when readable by others or empty). The desktop uses it with full scope. Every `/v1` route requires a bearer token except `/healthz` and `/v1/pair`.
- Implemented: device pairing (migration `0007_devices.sql`). `POST /v1/pairing-codes` (write scope) issues an 8-character code valid 5 min for one scope; `POST /v1/pair` redeems it once and returns a 256-bit device token. Only SHA-256 digests of codes and tokens are stored. `GET /v1/devices` lists devices with last access; `DELETE /v1/devices/{id}` revokes at once. Scopes: `full`, `approve_only` (approvals only), `read` (reads only); agent, memory and device mutations need `full`. `/v1/pair` allows 5 attempts per minute per TCP origin (`RemoteAddr`), then 429.
- Not implemented: QR code, mobile app, administrator password on the VPS node, storage of the desktop token in the vault (F1-T10), rate limit that is aware of a reverse proxy (behind one, all clients share the proxy's bucket), owner accounts (single `owner`).
