# Module: desktop-electron

Phase 1 (base) · Phases 4–5 (ADE, browser) · `apps/desktop` · ADR 0003

## Goal
Desktop app for Windows, macOS and Linux. A shell around the local `calango`; also connects to paired remote nodes.

## Structure
```
apps/desktop/
├── src/main/        main process: window, tray, update, calango sidecar
├── src/preload/     secure bridge (contextIsolation, no nodeIntegration)
└── src/renderer/    React + TS + Vite
```

## Rules
- The main process starts `calango` (binary packaged per platform), waits for `/healthz`, restarts with backoff if it dies, and shows the state in the tray.
- The renderer uses only `packages/sdk-ts`. The desktop token comes through the preload and is never kept in `localStorage`.
- Node selector: "This computer" and paired remote nodes.
- Electron security: `contextIsolation: true`, `sandbox: true`, strict CSP, no `remote`.
- Global shortcut to open the quick conversation.
- Distribution: signed NSIS (Windows), notarized DMG (macOS), `.deb` and AppImage (Linux). Signed auto-update.

## Screens in phase 1
- Sidebar (DMs, agents, routines, "For you"), conversation with steps, approval card, agent/routine editors, usage/costs, settings and audit.

## Acceptance
- Killing the `calango` process → the app shows "restarting" and comes back on its own within 10 s.
- The Windows installer does not trigger a SmartScreen warning because of missing signing.

## Current implementation status

- Agents, conversations, approvals, routines, usage, settings and audit screens are implemented. Fake-provider E2E covers chat, routine and usage; approval-card dispatch has a mocked renderer smoke.
- Electron launches the local daemon, creates a tray menu and registers `CommandOrControl+Shift+Space`. On Linux, startup-at-login uses an XDG autostart entry; isolated UI smoke verified create/remove.
- Tray creation and shortcut registration did not log errors, but OS-level shortcut activation and tray-menu interaction were not exercised. Windows/macOS login behavior, signing and installers remain unverified.
- Devices screen (#64, F2 foundation ahead of the phase gate): paired-device list with revoke confirmation and "Pair a new device" with a locally generated QR, typed code, countdown and expiry. Payload format in `pairing-auth.md`. The main process exposes only a display label of the host (`getNodeInfo`, IPC `calango:node-info`) for the QR `node` field. Verified on Linux with Xvfb and Playwright against a live daemon (QR decoded from a screenshot and redeemed with `POST /v1/pair`); no phone app exists yet.
- Internal browser foundation (F5-T01, #91): "Browser" screen with tabs as `WebContentsView` and one persistent partition per agent; rules and verification in `browser.md`.
