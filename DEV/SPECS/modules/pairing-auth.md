# Module: pairing-auth

Phase 1 (local token) · Phase 2 (pairing) · Package `internal/auth` · Tables `devices`, `pairing_codes`

## Goal
Only authorized devices talk to a node.

## Pairing flow
1. On the desktop (or in the VPS log): "Pair device" generates an 8-character code + QR, valid for 5 min, single use. Stored as a hash.
2. The app sends the code to `POST /v1/pair` through the transport.
3. The node returns a device token (random, 256 bits), stored in the phone's keychain. On the node, only the hash is kept.
4. A device list with last access; revoking invalidates it immediately.

## QR payload (version 1)

The desktop "Devices" screen encodes the pairing code in a QR as a URI. The phone app (#67) reads it and calls `POST /v1/pair`. Reference implementation and tests: `apps/desktop/src/shared/pairing.ts`, `apps/desktop/test/pairing.test.cjs`.

```
calango://pair?v=1&code=ABCD2345&scope=approve_only&exp=1764000300&node=meu-pc
```

The query is `application/x-www-form-urlencoded` (UTF-8; `+` or `%20` is a space).

| Field | Required | Meaning |
|---|---|---|
| `v` | yes | Payload version. Readers refuse any value other than `1` (`unsupported_version`). |
| `code` | yes | The single-use code from `POST /v1/pairing-codes`: 8 characters from `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`. Readers normalize it like `core.NormalizePairingCode` (drop spaces and dashes, upper-case). |
| `scope` | yes | `full`, `approve_only` or `read`: the scope the code grants. The app shows it before pairing and checks it against the `scope` in the `POST /v1/pair` response. |
| `exp` | yes | Expiry in Unix seconds (UTC), from `expires_at`. Lets the app say "expired" without a request; the node remains the authority. |
| `node` | no | Display label of the node (the desktop sends the host name, at most 64 characters, control characters removed). Not an identity and not trusted. |
| `addr` | no | Address of the node on the remote transport. Its format depends on the transport decision (#43/#65); the desktop does not emit it yet, because the daemon listens on loopback only. Opaque printable ASCII, at most 255 characters. |
| `fp` | no | Fingerprint of the node's transport key, so the app can confirm it reached the right node. Defined with #43/#65; not emitted yet. Opaque printable ASCII, at most 128 characters. |

Rules:
- The payload never carries the local API token, a device token, a vault value or any other long-lived secret. The only secret is the code, which is single-use and lasts 5 minutes; anyone who photographs the QR in that window can pair once, so the screen shows it only on request and hides it on expiry or once a new device appears.
- Readers ignore unknown parameters within `v=1`; a change that readers must understand needs `v=2`.
- The QR is generated on the desktop (`qrcode-generator`, error correction M, quiet zone of 4 modules). No external service sees the code.
- Issuing a new code does not cancel the previous one: it stays valid until used or expired. A route that cancels a code would need an API and storage change (not in #64).

## Rules
- Every route requires `Authorization: Bearer <token>`, except `/healthz` and `/v1/pair`.
- `/v1/pair` has an attempt limit (5 per minute per origin).
- Token scopes: `full`, `approve_only` (for a watch, for example), `read`.
- On the first start of the VPS node, an administrator password is required; no default credential.

## Acceptance
- A code used twice fails the second time.
- A revoked token gets `unauthorized` on the next call.

## Status (V1 local)

- Implemented (V1 local): local bearer token (`internal/auth`), stored as the vault item `api-token` (ADR 0012). A legacy file `DataDir/api-token` (mode 0600, refused when readable by others or empty) is imported into the vault and deleted on start. The desktop uses it with full scope. `POST /v1/auth/rotate-token` (full scope) replaces it: the old token gets 401 on the next call and the new token is returned once. Every `/v1` route requires a bearer token except `/healthz` and `/v1/pair`.
- Implemented: device pairing (migration `0007_devices.sql`). `POST /v1/pairing-codes` (write scope) issues an 8-character code valid 5 min for one scope; `POST /v1/pair` redeems it once and returns a 256-bit device token. Only SHA-256 digests of codes and tokens are stored. `GET /v1/devices` lists devices with last access; `DELETE /v1/devices/{id}` revokes at once. Scopes: `full`, `approve_only` (approvals only), `read` (reads only); agent, memory and device mutations need `full`. `/v1/pair` allows 5 attempts per minute per origin, then 429. The origin is the TCP peer, or the `X-Forwarded-For` client when the peer is listed in `CALANGO_TRUSTED_PROXIES` (`config.md`).
- Implemented (F2 foundation, #64, ahead of the phase gate): desktop "Devices" screen. It lists devices (name, platform, scope, pairing date, last use, status), revokes with an inline confirmation, and pairs a new device: choose the scope, generate a code, show the QR (payload above) and the code as text, a 5-minute countdown, announcements at one minute left, on expiry and when a new device appears, and a button to generate another code. The API was already sufficient; no route, schema or migration changed. `packages/sdk-ts` gained `createPairingCode`, `listDevices` and `revokeDevice`.
- Not implemented: phone-side QR reading and keychain storage (mobile app #67), remote transport and the `addr`/`fp` fields (#43/#65), administrator password on the VPS node, a route that revokes the local `api-token` without replacing it, owner accounts (single `owner`).
