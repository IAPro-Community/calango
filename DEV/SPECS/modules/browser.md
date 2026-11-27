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
- **Quit**: tabs are closed in `before-quit`, while the window still exists, with `close({ waitForBeforeUnload: false })`; a live tab WebContents would otherwise keep the process alive after quit.
- **Downloads**: denied (`will-download` is cancelled). A controlled download flow (approval, target folder) is a later step.
- **Renderer**: "Browser" screen with agent selector, tab strip, address bar and back/forward/reload, and a two-step "Clear profile". The renderer sends only the viewport rectangle and tab commands; page content never reaches it. Texts in `packages/i18n`.

Extension points left for the next tasks:
- F5-T02 (#92): `BrowserManager.webContentsFor(tabId)` (main process only, never over IPC) to attach CDP and implement take over / hand back without reload.
- F5-T03 (#93): `BrowserManager.observe({ tabCreated, tabLoaded, tabClosed })` to inspect loads and signal barriers to the agent without quoting page text.
- F4-T06 (#87): done, see "Project preview" below.

Verification: `npm run test:main -w apps/desktop` (`test/browser.test.cjs`, with Electron fakes) covers partition isolation, permission denial, protocol gate, window-open handler, downloads, clear profile and view placement. A manual Electron smoke (xvfb, real `session`/`WebContentsView`) confirmed: a cookie set in agent A's partition is absent in agent B's; `getUserMedia` → `NotAllowedError`, notifications and geolocation `denied`; `require`/`process` undefined in the page; `file:` link, custom-scheme navigation and a 302 to a custom scheme are blocked; `window.open` of http opens a tab and of `file:` does nothing; clear profile leaves zero cookies and closes only that agent's tabs. Not yet verified: the full app UI with the daemon running, Windows and macOS.

## Project preview (F4-T06 foundation, desktop)

F4-T06 (#87) lands ahead of the F4 acceptance on top of F5-T01. Code in `apps/desktop/src/main/browser/preview.ts` and `BrowserManager.openPreview`.

- **Detection**: "Detect local servers" probes a fixed short list of ports (3000, 3001, 4173, 4200, 5000, 5173, 5174, 8000, 8080, 8888; at most 32 for a caller-supplied list) with one `GET /` each, in parallel, 800 ms timeout. It connects only to the literal addresses `127.0.0.1` and then `::1` (never resolves a name, never another machine), sends `Host: localhost:<port>`, does not follow redirects and reads at most 64 KiB. It reports port, origin, HTTP status, a plain-text `<title>` (entities decoded, control and bidi characters removed, 120 characters) and a best-effort kind (vite, next, angular, webpack, express, python, http). The Calango daemon's own port is excluded from detection and from opening.
- **Target**: a port, `localhost:PORT` or `http://localhost|127.0.0.1|[::1]:PORT/path`. After WHATWG parsing the host must be exactly `localhost`, `127.0.0.1` or `[::1]`, the scheme plain `http` and there must be no userinfo. `0.0.0.0`, `[::]`, LAN and public IPs, other `127.x` addresses, IPv4-mapped IPv6, `*.localhost`, `localhost.evil.com`, DNS names that resolve to loopback (`localtest.me`) and `https` are refused. `localhost` is accepted because Chromium resolves it to loopback itself.
- **Partition**: preview tabs use their own persistent partition, `persist:calango-preview-<32 hex>` (HMAC under the same per-install key), never an agent's. Their owner id in the bridge is `@preview`, which can never be an agent id. Hook for the project registry (#82) and per-task worktrees (#83): `openPreview(target, { projectId })` gives each project its own preview partition; today every preview shares the one without project.
- **Containment (chosen: block, with an explicit way out)**: a preview tab stays on its origin. `localhost`, `127.0.0.1` and `[::1]` on the same port count as the same origin; any other host, port or scheme is outside. Outside navigations (`will-navigate`, main-frame `will-redirect`, `window.open`, and the address bar) are cancelled; the tab stays where it was. Same-origin popups open as preview tabs with the same origin. A second layer, `webRequest.onBeforeRequest` on the preview partition only, cancels any main-frame request that is not from a known preview tab or leaves that tab's origin, whatever started it. Subresources and iframes are not filtered (dev pages load CDN scripts and fonts).
- **Leaving the preview**: a blocked http(s) URL is shown in the panel with "Open in the system browser". Only a URL that a preview tab was just kept from (at most the last 20) can be opened, once each, through `shell.openExternal`; the person's click is the confirmation. Nothing opens outside on its own.
- **Same hardening as agent tabs**: sandbox, no Node, no preload, every permission and device denied, downloads cancelled, `file:`/custom schemes refused.
- **Auto reload (optional, off by default)**: when the dev server is down (connection refused/reset/closed/aborted, unreachable, empty response, timeout) the main-frame load is retried with backoff from 2 s up to 10 s until it answers. Loads cancelled by the containment or replaced by a newer navigation are never retried. The pending retry is cancelled when the tab closes. HMR of the dev server itself keeps working as usual.
- **Renderer**: on the Browser screen, the profile selector has a "Project preview" entry (also usable with no agents or without the daemon) with "Detect local servers", the list of servers (port, title, kind, status, "Open preview"), a manual port field and the auto reload checkbox. Texts in `packages/i18n`.
- **Agent hook**: main-process callers can use `detectDevServers(ports)` / `probePort` and `BrowserManager.openPreview(target, options)` directly; no agent tool is exposed yet.

Not covered yet: https dev servers (self-signed certificates), a preview of a server on the VPS node, binding a preview to a project or worktree (waits for #82/#83), and filtering subresources.

Verification: `test/preview.test.cjs` covers loopback origin validation (0.0.0.0, external IPs, `localhost.evil.com`, userinfo, other schemes, DNS names), containment of navigations, redirects, popups and the address bar, the network-layer filter, the system-browser gate, auto reload with fake timers, the separate partition and clear profile, and the probe against a real local HTTP server, a closed port, a server that never answers and an IPv6-only server. A manual smoke of the real app (xvfb + Playwright, Electron 45) detected a local `python -m http.server` on 5173 with its title, opened the preview in `calango-preview-<hex>`, kept the tab on its URL after a click to `https://example.com`, a `location.href` to the daemon port and a `window.open` to another site, showed the notice and opened the URL outside only after the click (stubbed `shell.openExternal`), followed a same-origin link, refused `0.0.0.0:5173` in the port field, and `app.quit()` exited with code 0 with preview tabs open.
