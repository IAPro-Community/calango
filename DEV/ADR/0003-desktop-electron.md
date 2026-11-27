# ADR 0003 — Electron desktop with `calango` as a child process

Status: accepted · 2026-11-23 (replaces the initial Wails proposal)

## Context
The desktop app needs a real embedded browser (tabs the agent drives and the person can take over), an editor and a terminal for the ADE, a tray icon, auto-update and an installer for Windows, macOS and Linux.

## Decision
Electron. The main process starts the `calango` binary as a child process, watches its health and restarts it if it dies. The renderer (React + TypeScript + Vite) talks to `calango` through the local API, like any other client. Internal browser with `WebContentsView`; the agent drives it through the Electron debugging protocol.

## Alternatives
- Wails (Go + system WebView): smaller, but the system WebView does not offer full Chromium tabs or uniform CDP control across platforms.
- Tauri: same WebView limitation.

## Consequences
- Larger installer (~100 MB). Accepted.
- Signed installers from the first public release (Windows code signing, macOS notarization).
- `calango` must be packaged per platform together with the app.

## Implementation addendum (2026-11-25, F5-T01)

- Each agent gets one persistent partition named `persist:calango-agent-<HMAC-SHA256(per-install key, agent id), 32 hex>`; the key lives in the Electron user data folder (mode 0600). Tabs are `WebContentsView`s with no preload, sandboxed, and limited to http(s).
- Web permissions and downloads are denied by default; popups become tabs of the same agent. Details in `DEV/SPECS/modules/browser.md`.
