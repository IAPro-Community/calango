// Security hooks for the internal browser. Only Electron *types* are imported, so the
// main-process tests can drive these functions with fakes under plain Node.
import type { Session, WebContents, WebPreferences } from 'electron';
import { decideWindowOpen, isNavigableUrl, permissionAllowed } from './policy';

/**
 * webPreferencesFor returns the preferences of a tab's WebContentsView. Web content gets no
 * preload, no Node, its own sandboxed renderer and only its agent's partition.
 */
export function webPreferencesFor(partition: string): WebPreferences {
  return {
    partition,
    sandbox: true,
    contextIsolation: true,
    nodeIntegration: false,
    nodeIntegrationInSubFrames: false,
    nodeIntegrationInWorker: false,
    webviewTag: false,
    webSecurity: true,
    allowRunningInsecureContent: false,
    navigateOnDragDrop: false,
    safeDialogs: true,
    spellcheck: false,
  };
}

/**
 * hardenSession denies every permission and device request and cancels every download in
 * one agent partition. Run it once per partition, before the first tab loads.
 */
export function hardenSession(session: Session, onDownloadBlocked?: (url: string) => void): void {
  session.setPermissionRequestHandler((_webContents, permission, callback) => callback(permissionAllowed(permission)));
  session.setPermissionCheckHandler((_webContents, permission) => permissionAllowed(permission));
  session.setDevicePermissionHandler(() => false);
  session.setBluetoothPairingHandler((_details, callback) => callback({ confirmed: false }));
  // No display-media handler is set: without one, getDisplayMedia is refused.
  session.on('select-hid-device', (event, _details, callback) => {
    event.preventDefault();
    callback(null);
  });
  session.on('select-serial-port', (event, _ports, _webContents, callback) => {
    event.preventDefault();
    callback('');
  });
  session.on('select-usb-device', (event, _details, callback) => {
    event.preventDefault();
    callback();
  });
  // Downloads are denied in this foundation. A controlled download flow (approval, target
  // folder) is a later step; the hook stays here.
  session.on('will-download', (event, item) => {
    event.preventDefault();
    onDownloadBlocked?.(item.getURL());
  });
}

export interface WebContentsHooks {
  /** A popup or target=_blank link asked for this URL; open it as a controlled tab. */
  openTab(url: string): void;
  /** A navigation was refused by the protocol gate. */
  blocked?(url: string): void;
}

/**
 * hardenWebContents wires the per-tab rules: popups become tabs of the same agent, top-level
 * navigations and redirects outside http(s) are cancelled, and <webview> is refused.
 */
export function hardenWebContents(contents: WebContents, hooks: WebContentsHooks): void {
  contents.setWindowOpenHandler((details) => {
    if (decideWindowOpen(details.url) === 'tab') {
      hooks.openTab(details.url);
    } else {
      hooks.blocked?.(details.url);
    }
    return { action: 'deny' };
  });
  contents.on('will-navigate', (event) => {
    if (!isNavigableUrl(event.url)) {
      event.preventDefault();
      hooks.blocked?.(event.url);
    }
  });
  contents.on('will-redirect', (event) => {
    if (event.isMainFrame && !isNavigableUrl(event.url)) {
      event.preventDefault();
      hooks.blocked?.(event.url);
    }
  });
  contents.on('will-attach-webview', (event) => {
    event.preventDefault();
  });
}
