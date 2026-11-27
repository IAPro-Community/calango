import {
  app,
  BrowserWindow,
  globalShortcut,
  ipcMain,
  Menu,
  nativeImage,
  protocol,
  session,
  Tray,
  WebContentsView,
  type IpcMainInvokeEvent,
} from 'electron';
import { createWriteStream, mkdirSync } from 'node:fs';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { channels, type BrowserTab, type DaemonState } from '../shared/ipc';
import { BrowserManager } from './browser/manager';
import { parseBounds } from './browser/policy';
import { loadProfileKey } from './browser/profileKey';
import { Daemon } from './daemon';
import { messages, type Locale } from './trayMessages';
import { linuxAutostartEnabled, linuxAutostartFilePath, setLinuxAutostart } from './startup';

const APP_SCHEME = 'app';
const APP_ORIGIN = `${APP_SCHEME}://calango`;
// Compiled to dist/main/index.js, so the repository root is four levels up.
const repoRoot = path.resolve(__dirname, '..', '..', '..', '..');
const rendererRoot = path.resolve(__dirname, '..', 'renderer');

const addr = loopbackAddr(process.env.CALANGO_ADDR ?? '127.0.0.1:7317');
const binaryName = process.platform === 'win32' ? 'calango.exe' : 'calango';
// The installed app carries the daemon in resources/bin (apps/desktop/electron-builder.yml).
const binary =
  process.env.CALANGO_BIN ??
  (app.isPackaged ? path.join(process.resourcesPath, 'bin', binaryName) : path.join(repoRoot, 'bin', binaryName));

// The daemon's CORS allowlist names this origin (cmd/calango). connect-src lists only
// the daemon, so the renderer cannot reach any other host.
const contentSecurityPolicy = [
  "default-src 'self'",
  "script-src 'self'",
  "style-src 'self'",
  "img-src 'self' data:",
  "font-src 'self'",
  `connect-src 'self' http://${addr}`,
  "object-src 'none'",
  "base-uri 'none'",
  "frame-ancestors 'none'",
].join('; ');

const mimeTypes: Record<string, string> = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.woff2': 'font/woff2',
};

let mainWindow: BrowserWindow | null = null;
let tray: Tray | null = null;
let shuttingDown = false;
let browser: BrowserManager | null = null;

const daemon = new Daemon({
  binary,
  addr,
  onState: broadcastState,
  output: app.isPackaged ? daemonLogFile() : undefined,
});

// daemonLogFile opens the log file of an installed app, which has no console. The daemon
// never logs the API token or vault values, so the file holds no secrets.
function daemonLogFile(): NodeJS.WritableStream {
  const logsDir = app.getPath('logs');
  mkdirSync(logsDir, { recursive: true });
  const stream = createWriteStream(path.join(logsDir, 'daemon.log'), { flags: 'a' });
  stream.on('error', (err) => console.error(`Calango could not write the daemon log: ${err.message}`));
  return stream;
}

function loopbackAddr(value: string): string {
  const match = /^127\.0\.0\.1:(\d{1,5})$/.exec(value);
  if (match === null || Number(match[1]) < 1 || Number(match[1]) > 65535) {
    throw new Error(`CALANGO_ADDR must be 127.0.0.1:<port>, got ${value}`);
  }
  return value;
}

function broadcastState(state: DaemonState): void {
  for (const win of BrowserWindow.getAllWindows()) {
    if (!win.isDestroyed()) {
      win.webContents.send(channels.stateChanged, state);
    }
  }
}

// serveRenderer maps app://calango/... to files under dist/renderer.
async function serveRenderer(request: Request): Promise<Response> {
  try {
    const url = new URL(request.url);
    if (url.hostname !== 'calango') {
      return new Response('not found', { status: 404 });
    }
    const rel = url.pathname === '/' ? '/index.html' : decodeURIComponent(url.pathname);
    const file = path.resolve(rendererRoot, `.${rel}`);
    if (!file.startsWith(rendererRoot + path.sep)) {
      return new Response('forbidden', { status: 403 });
    }
    const body = await readFile(file);
    return new Response(body, {
      headers: {
        'Content-Type': mimeTypes[path.extname(file)] ?? 'application/octet-stream',
        'Content-Security-Policy': contentSecurityPolicy,
      },
    });
  } catch {
    return new Response('not found', { status: 404 });
  }
}

// assertTrusted rejects IPC from any frame that is not the renderer page.
function assertTrusted(event: IpcMainInvokeEvent): void {
  const url = event.senderFrame?.url ?? '';
  if (!url.startsWith(`${APP_ORIGIN}/`)) {
    throw new Error('untrusted IPC sender');
  }
}

function createBrowser(win: BrowserWindow, profileKey: Buffer): BrowserManager {
  return new BrowserManager({
    profileKey,
    fromPartition: (partition) => session.fromPartition(partition),
    createView: (options) => new WebContentsView(options),
    host: {
      add: (view) => {
        if (!win.isDestroyed()) win.contentView.addChildView(view);
      },
      remove: (view) => {
        if (!win.isDestroyed()) win.contentView.removeChildView(view);
      },
    },
    onTabsChanged: (agentId: string, tabs: BrowserTab[]) => {
      if (!win.isDestroyed()) {
        win.webContents.send(channels.browserTabsChanged, { agentId, tabs });
      }
    },
  });
}

function requireBrowser(event: IpcMainInvokeEvent): BrowserManager {
  assertTrusted(event);
  if (browser === null) {
    throw new Error('browser is not ready');
  }
  return browser;
}

// The renderer can only drive the browser through these handlers; tab content has no
// preload and fails assertTrusted, so a web page cannot reach them.
function registerBrowserIpc(): void {
  ipcMain.handle(channels.browserList, (event, agentId: unknown) => requireBrowser(event).listTabs(agentId));
  ipcMain.handle(channels.browserFocusAgent, (event, agentId: unknown) => requireBrowser(event).focusAgent(agentId));
  ipcMain.handle(channels.browserOpenTab, (event, agentId: unknown, url: unknown) =>
    requireBrowser(event).openTab(agentId, url),
  );
  ipcMain.handle(channels.browserCloseTab, (event, tabId: unknown) => requireBrowser(event).closeTab(tabId));
  ipcMain.handle(channels.browserActivate, (event, tabId: unknown) => requireBrowser(event).activate(tabId));
  ipcMain.handle(channels.browserNavigate, (event, tabId: unknown, address: unknown) =>
    requireBrowser(event).navigate(tabId, address),
  );
  ipcMain.handle(channels.browserBack, (event, tabId: unknown) => requireBrowser(event).goBack(tabId));
  ipcMain.handle(channels.browserForward, (event, tabId: unknown) => requireBrowser(event).goForward(tabId));
  ipcMain.handle(channels.browserReload, (event, tabId: unknown) => requireBrowser(event).reload(tabId));
  ipcMain.handle(channels.browserSetBounds, (event, bounds: unknown) => {
    const manager = requireBrowser(event);
    const size = mainWindow?.getContentBounds() ?? { width: 0, height: 0 };
    manager.setBounds(bounds === null ? null : parseBounds(bounds, size));
  });
  ipcMain.handle(channels.browserClearProfile, (event, agentId: unknown) =>
    requireBrowser(event).clearProfile(agentId),
  );
}

function linuxAutostartPath(): string {
  return linuxAutostartFilePath(process.env.XDG_CONFIG_HOME, app.getPath('home'));
}

async function launchAtLoginEnabled(): Promise<boolean> {
  if (process.platform === 'linux') {
    return linuxAutostartEnabled(linuxAutostartPath());
  }
  return app.getLoginItemSettings().openAtLogin;
}

async function applyLaunchAtLogin(enabled: boolean): Promise<boolean> {
  if (process.platform === 'linux') {
    const args = app.isPackaged ? [] : [app.getAppPath()];
    const filePath = linuxAutostartPath();
    await setLinuxAutostart(filePath, enabled, process.execPath, args);
    return linuxAutostartEnabled(filePath);
  }
  app.setLoginItemSettings({ openAtLogin: enabled });
  return app.getLoginItemSettings().openAtLogin;
}

function registerIpc(): void {
  ipcMain.handle(channels.connection, (event) => {
    assertTrusted(event);
    return daemon.connectionOrThrow();
  });
  ipcMain.handle(channels.state, (event) => {
    assertTrusted(event);
    return daemon.getState();
  });
  ipcMain.handle(channels.getLoginItemSettings, async (event) => {
    assertTrusted(event);
    return { openAtLogin: await launchAtLoginEnabled() };
  });
  ipcMain.handle(channels.setLaunchAtLogin, async (event, enabled: unknown) => {
    assertTrusted(event);
    if (typeof enabled !== 'boolean') {
      throw new Error('launch-at-login preference must be a boolean');
    }
    const openAtLogin = await applyLaunchAtLogin(enabled);
    if (openAtLogin !== enabled) {
      throw new Error('operating system did not apply the launch-at-login preference');
    }
    return { openAtLogin };
  });
}

function showWindow(section?: 'conversations'): void {
  if (mainWindow === null || mainWindow.isDestroyed()) {
    return;
  }
  if (section === 'conversations') {
    mainWindow.webContents.send(channels.quickConversation);
  }
  if (mainWindow.isMinimized()) {
    mainWindow.restore();
  }
  mainWindow.show();
  mainWindow.focus();
}

function localeForApp(): Locale {
  const locale = app.getLocale().toLowerCase();
  if (locale.startsWith('pt')) {
    return 'pt-BR';
  }
  if (locale.startsWith('es')) {
    return 'es';
  }
  return 'en';
}

function createTray(): void {
  const icon = nativeImage.createFromDataURL(
    `data:image/svg+xml;charset=utf-8,${encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 32 32"><rect width="32" height="32" rx="7" fill="#FDD838"/><path d="M7 21c2-3 5-4 8-4 1-4 4-7 9-7l3 2-3 2c1 2 2 4 3 6l-3 2-3-3-5 1-3 5-3-1 2-5-4 2z" fill="#42912E" stroke="#22180F" stroke-width="1.2" stroke-linejoin="round"/><circle cx="23" cy="12" r="1" fill="#22180F"/></svg>')}`,
  );
  const t = messages[localeForApp()];
  tray = new Tray(icon);
  tray.setToolTip('Calango');
  tray.setContextMenu(
    Menu.buildFromTemplate([
      { label: t.trayOpen, click: () => showWindow() },
      { label: t.trayQuickConversation, click: () => showWindow('conversations') },
      { type: 'separator' },
      { label: t.trayQuit, click: () => app.quit() },
    ]),
  );
  tray.on('click', () => showWindow());
  tray.on('double-click', () => showWindow());
}

function registerQuickConversationShortcut(): void {
  const registered = globalShortcut.register('CommandOrControl+Shift+Space', () => showWindow('conversations'));
  if (!registered) {
    console.warn('Calango could not register the quick conversation shortcut');
  }
}

function createWindow(): BrowserWindow {
  const win = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 960,
    minHeight: 640,
    title: 'Calango',
    show: false,
    webPreferences: {
      preload: path.join(__dirname, '..', 'preload', 'index.js'),
      contextIsolation: true,
      sandbox: true,
      nodeIntegration: false,
    },
  });
  win.once('ready-to-show', () => win.show());
  win.on('close', (event) => {
    if (!shuttingDown) {
      event.preventDefault();
      win.hide();
    }
  });
  win.webContents.setWindowOpenHandler(() => ({ action: 'deny' }));
  win.webContents.session.setPermissionRequestHandler((_webContents, _permission, callback) => callback(false));
  win.webContents.on('will-navigate', (event, url) => {
    if (!url.startsWith(`${APP_ORIGIN}/`)) {
      event.preventDefault();
    }
  });
  void win.loadURL(`${APP_ORIGIN}/`);
  return win;
}

if (!app.requestSingleInstanceLock()) {
  app.quit();
} else {
  // Must run before the app is ready.
  protocol.registerSchemesAsPrivileged([
    { scheme: APP_SCHEME, privileges: { standard: true, secure: true, supportFetchAPI: true } },
  ]);

  app.on('second-instance', () => showWindow());

  app.on('activate', () => showWindow());

  app.whenReady().then(async () => {
    protocol.handle(APP_SCHEME, serveRenderer);
    registerIpc();
    registerBrowserIpc();
    mainWindow = createWindow();
    try {
      const profileKey = await loadProfileKey(path.join(app.getPath('userData'), 'browser-profile.key'));
      browser = createBrowser(mainWindow, profileKey);
    } catch {
      console.warn('Calango could not prepare the internal browser profile key');
    }
    createTray();
    registerQuickConversationShortcut();
    void daemon.start();
  });

  app.on('will-quit', () => {
    browser?.dispose();
    globalShortcut.unregisterAll();
    tray?.destroy();
  });

  app.on('window-all-closed', () => {
    // The tray keeps the app alive after the main window is hidden/closed.
  });

  app.on('before-quit', (event) => {
    if (shuttingDown) {
      return;
    }
    shuttingDown = true;
    event.preventDefault();
    // Close browser tabs while the window still exists; their WebContents would otherwise
    // keep the process alive after quit.
    browser?.dispose();
    void daemon.stop().finally(() => app.quit());
  });
}
