import { app, BrowserWindow, globalShortcut, ipcMain, Menu, protocol, Tray, type IpcMainInvokeEvent } from 'electron';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { channels, type DaemonState } from '../shared/ipc';
import { Daemon } from './daemon';
import { messages, type Locale } from './trayMessages';
import { fallbackTrayIcon, loadTrayIcon, TRAY_ICON_PNG } from './trayIcon';
import { linuxAutostartEnabled, linuxAutostartFilePath, setLinuxAutostart } from './startup';

const APP_SCHEME = 'app';
const APP_ORIGIN = `${APP_SCHEME}://calango`;
// Compiled to dist/main/index.js, so the repository root is four levels up.
const repoRoot = path.resolve(__dirname, '..', '..', '..', '..');
const rendererRoot = path.resolve(__dirname, '..', 'renderer');

const addr = loopbackAddr(process.env.CALANGO_ADDR ?? '127.0.0.1:7317');
const binary = process.env.CALANGO_BIN ?? path.join(repoRoot, 'bin', 'calango');

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

const daemon = new Daemon({
  binary,
  addr,
  onState: broadcastState,
});

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
  // Tray needs a raster image on Windows and macOS; SVG data URLs only render
  // on Linux. Missing PNG falls back to the SVG so a dev run still shows a tray.
  const icon = loadTrayIcon({ pngPath: path.join(__dirname, '..', 'renderer', TRAY_ICON_PNG) });
  if (icon === null) {
    console.warn('Calango tray icon asset missing; using the SVG fallback');
  }
  const t = messages[localeForApp()];
  tray = new Tray(icon ?? fallbackTrayIcon());
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

// notifyShortcutUnavailable tells the user the accelerator is taken. It must
// not block start-up: the window is created and the daemon started regardless.
function notifyShortcutUnavailable(): void {
  if (mainWindow === null || mainWindow.isDestroyed() || !mainWindow.isVisible()) {
    showWindow();
  }
  const t = messages[localeForApp()];
  if (mainWindow !== null && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send(channels.shortcutUnavailable, t.trayShortcutUnavailable);
  }
}

function registerQuickConversationShortcut(): void {
  const registered = globalShortcut.register('CommandOrControl+Shift+Space', () => showWindow('conversations'));
  if (!registered) {
    // The shortcut is an accelerator, not a feature: keep the app usable and
    // tell the user instead of failing silently.
    console.warn('Calango could not register the quick conversation shortcut');
    notifyShortcutUnavailable();
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

  app.whenReady().then(() => {
    protocol.handle(APP_SCHEME, serveRenderer);
    registerIpc();
    mainWindow = createWindow();
    createTray();
    registerQuickConversationShortcut();
    void daemon.start();
  });

  app.on('will-quit', () => {
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
    void daemon.stop().finally(() => app.quit());
  });
}
