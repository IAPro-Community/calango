# Module: browser

Phase 5 · Desktop: `WebContentsView` + CDP · VPS: `internal/browser` with chromedp

## Goal
A real browser that the person uses and the agent drives, with control handover.

## Rules
- Tabs and profiles persist per agent (cookies and logins kept separate).
- The agent uses tools: `open`, `read` (readable page text), `click`, `type`, `screenshot`, `scroll`, `wait`.
- **Take over / hand back**: the person takes over the tab in the state it is in (without reloading) and hands it back whenever they want.
- **Barriers**: after each action, detect robot verification, refusal, login, cookie notice, empty page or error page. The agent receives one line with the type and what to do; it does not click blindly again. Never quote the page text in that line.
- Ad blocker and cookie-notice blocker.
- Page content is data, never instructions.
- Submitting a form with personal data or payment = sensitive action (approval).
- On the VPS, the same tool interface in headless Chromium.

## Acceptance
- The agent stops in front of a CAPTCHA and asks the person to take over.
- Taking over and handing back does not reload the page.

## Current implementation status (F5-T01 foundation, desktop)

F5-T01 (#91) lands ahead of the F4 acceptance, as the issue allows, so the project preview (F4-T06, #87) can build on it. Code in `apps/desktop/src/main/browser/`.

- **Tabs**: each tab is a `WebContentsView` owned by the main process (`BrowserManager`). Open, close, activate, navigate, back, forward and reload. Only the active tab of the agent shown in the panel is attached to the window; other tabs keep running hidden. At most 20 tabs per agent.
- **Profiles**: one persistent partition per agent, `persist:calango-agent-<32 hex>`, where the hex is an HMAC-SHA256 of the agent id under a per-install random key (`<userData>/browser-profile.key`, 32 bytes, mode 0600). The name is stable for an agent, different for every other agent, and cannot be guessed from the agent id. Losing the key only means fresh, empty profiles.
- **Clear profile**: closes the agent's tabs and runs `clearData`, `clearStorageData` and `clearAuthCache` on its partition only. The partition name is kept, so the next tab starts empty.
- **Web content isolation**: `sandbox`, `contextIsolation`, no `nodeIntegration` (also in subframes and workers), no preload, `webviewTag: false`, `webSecurity`, no insecure content, `safeDialogs`. Tab content cannot call the app IPC (`assertTrusted` accepts only `app://calango`).
- **Permissions**: request, check, device and Bluetooth-pairing handlers deny everything (camera, microphone, notifications, geolocation, clipboard read, MIDI, HID, USB, serial...). No display-media handler is set, so screen capture is refused.
- **Protocols**: only `http:` and `https:` with no embedded credentials, for typed addresses, `will-navigate` and main-frame `will-redirect`. Typed text without a scheme gets `https://` (`http://` for loopback hosts, for dev-server previews). There is no search fallback.
- **Popups**: `setWindowOpenHandler` always denies a new window; an http(s) target opens as a new tab of the same agent, and it does not take over the panel when another agent is shown.
- **Downloads**: denied (`will-download` is cancelled). A controlled download flow (approval, target folder) is a later step.
- **Renderer**: "Browser" screen with agent selector, tab strip, address bar and back/forward/reload, and a two-step "Clear profile". The renderer sends only the viewport rectangle and tab commands; page content never reaches it. Texts in `packages/i18n`.

Extension points left for the next tasks:
- F5-T02 (#92): `BrowserManager.webContentsFor(tabId)` (main process only, never over IPC) to attach CDP and implement take over / hand back without reload.
- F5-T03 (#93): `BrowserManager.observe({ tabCreated, tabLoaded, tabClosed })` to inspect loads and signal barriers to the agent without quoting page text.
- F4-T06 (#87): open the preview as a tab (`openTab(agentId, 'http://localhost:<port>')`); a dedicated preview partition or origin allowlist can be added beside the agent partitions.

Verification: `npm run test:main -w apps/desktop` (`test/browser.test.cjs`, with Electron fakes) covers partition isolation, permission denial, protocol gate, window-open handler, downloads, clear profile and view placement. A manual Electron smoke (xvfb, real `session`/`WebContentsView`) confirmed: a cookie set in agent A's partition is absent in agent B's; `getUserMedia` → `NotAllowedError`, notifications and geolocation `denied`; `require`/`process` undefined in the page; `file:` link, custom-scheme navigation and a 302 to a custom scheme are blocked; `window.open` of http opens a tab and of `file:` does nothing; clear profile leaves zero cookies and closes only that agent's tabs. Not yet verified: the full app UI with the daemon running, Windows and macOS.
