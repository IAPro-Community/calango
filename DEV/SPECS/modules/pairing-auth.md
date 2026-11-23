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
