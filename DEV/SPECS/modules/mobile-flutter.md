# Module: mobile-flutter

Phase 2 · `apps/mobile` · ADR 0004

## Goal
Remote control of Calango on the phone: see, approve, chat, schedule.

## Screens
1. Welcome + pair (camera for QR or code). QR payload: `pairing-auth.md` (version 1, `calango://pair?...`).
2. Nodes: PC and VPS, online/offline, since when, what is waiting for each.
3. "For you": approvals, questions, deliveries.
4. Conversations (DMs) with summarized steps.
5. Approval card with action, target, arguments, approve/refuse/always approve.
6. Routines: list, run now, pause.
7. Tasks (simplified board).
8. Settings: devices, plan, notifications.
9. PC screen control (phase 5, paid).

## Rules
- The client only via `packages/sdk-dart` (`CalangoClient`, generated from the OpenAPI). Theme only via `packages/theme-flutter` (`CalangoTheme.light()` / `dark()`, composed from generated tokens). Both exist since F2-T03 (ADR 0024); the WebSocket events client and font bundling belong to the app.
- Token in the Keychain (iOS) / Keystore (Android).
- A push opens directly on the item (approval, message).
- Works with more than one node; selector at the top.
- Purchase through the store's system; plan recognized in all clients.

## Acceptance
- Approve an action from the PC while on 4G, via Tailscale, in less than 3 taps from the push.
