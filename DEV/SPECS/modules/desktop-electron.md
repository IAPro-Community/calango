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
Sidebar (DMs, agents, routines, "For you"), conversation with steps, approval card, agent editor, routine editor, usage and costs, settings (you, agents, models, connections, vault).

## Acceptance
- Killing the `calango` process → the app shows "restarting" and comes back on its own within 10 s.
- The Windows installer does not trigger a SmartScreen warning because of missing signing.
