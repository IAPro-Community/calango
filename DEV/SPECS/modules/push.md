# Module: push

Phase 2 · `cmd/calango-cloud` (service `push`) + `internal/push` on the node

## Goal
Notify the phone even when the app is closed, without Calango seeing any content.

## Flow
1. The app registers the APNs/FCM token with Calango Cloud, linked to the device.
2. The node sends the relay: `{ device_id, kind, opaque_id }`. No text.
3. The relay sends APNs/FCM a generic notification ("Calango needs you") with the `opaque_id`.
4. The app opens and fetches the content from the node through the transport.

## Rules
- Authentication of the node with the relay by the account key.
- Limit per plan (Free limited, Pro reasonably unlimited).
- Grouping: several notifications within 30 s become one.

## Acceptance
- Inspecting the payload at the relay shows no conversation text (test).
