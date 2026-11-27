const assert = require('node:assert/strict');
const { EventEmitter } = require('node:events');
const { mkdtemp, readFile, rm, stat } = require('node:fs/promises');
const os = require('node:os');
const path = require('node:path');
const { test } = require('node:test');
const {
  AGENT_PARTITION_PREFIX,
  decideWindowOpen,
  isNavigableUrl,
  normalizeAddress,
  parseBounds,
  partitionForAgent,
} = require('../dist/main/browser/policy.js');
const { BrowserManager } = require('../dist/main/browser/manager.js');
const { loadProfileKey } = require('../dist/main/browser/profileKey.js');

const KEY = Buffer.alloc(32, 7);
const OTHER_KEY = Buffer.alloc(32, 9);

// Fakes of the Electron objects the manager touches. Only the methods used are modelled.
class FakeSession extends EventEmitter {
  constructor(partition) {
    super();
    this.partition = partition;
    this.cleared = [];
  }
  setPermissionRequestHandler(handler) {
    this.permissionRequest = handler;
  }
  setPermissionCheckHandler(handler) {
    this.permissionCheck = handler;
  }
  setDevicePermissionHandler(handler) {
    this.devicePermission = handler;
  }
  setBluetoothPairingHandler(handler) {
    this.bluetoothPairing = handler;
  }
  async clearData() {
    this.cleared.push('data');
  }
  async clearStorageData() {
    this.cleared.push('storage');
  }
  async clearAuthCache() {
    this.cleared.push('auth');
  }
}

class FakeWebContents extends EventEmitter {
  constructor() {
    super();
    this.loaded = [];
    this.destroyed = false;
    this.url = '';
    this.navigationHistory = {
      canGoBack: () => false,
      canGoForward: () => false,
      goBack: () => {},
      goForward: () => {},
    };
  }
  setWindowOpenHandler(handler) {
    this.windowOpen = handler;
  }
  loadURL(url) {
    this.loaded.push(url);
    this.url = url;
    return Promise.resolve();
  }
  reload() {}
  getURL() {
    return this.url;
  }
  getTitle() {
    return '';
  }
  isLoading() {
    return false;
  }
  isDestroyed() {
    return this.destroyed;
  }
  close(options) {
    this.closeOptions = options;
    this.destroyed = true;
  }
}

function setup(hostOverrides = {}) {
  const sessions = new Map();
  const views = [];
  const attached = new Set();
  const changes = [];
  const manager = new BrowserManager({
    profileKey: KEY,
    fromPartition: (partition) => {
      if (!sessions.has(partition)) sessions.set(partition, new FakeSession(partition));
      return sessions.get(partition);
    },
    createView: (options) => {
      const view = { options, webContents: new FakeWebContents(), bounds: null };
      view.setBounds = (bounds) => {
        view.bounds = bounds;
      };
      views.push(view);
      return view;
    },
    host: {
      add: (view) => attached.add(view),
      remove: (view) => attached.delete(view),
      ...hostOverrides,
    },
    onTabsChanged: (agentId, tabs) => changes.push({ agentId, tabs }),
  });
  return { manager, sessions, views, attached, changes };
}

function navigationEvent(url, isMainFrame = true) {
  return {
    url,
    isMainFrame,
    defaultPrevented: false,
    preventDefault() {
      this.defaultPrevented = true;
    },
  };
}

test('partition per agent is persistent, stable, distinct and not derived from the id alone', () => {
  const a = partitionForAgent('agent_A', KEY);
  const b = partitionForAgent('agent_B', KEY);
  assert.ok(a.startsWith('persist:'));
  assert.ok(a.startsWith(AGENT_PARTITION_PREFIX));
  assert.equal(a, partitionForAgent('agent_A', KEY));
  assert.notEqual(a, b);
  assert.notEqual(a, partitionForAgent('agent_A', OTHER_KEY));
  assert.ok(!a.includes('agent_A'));
  assert.match(a, /^persist:calango-agent-[0-9a-f]{32}$/);
  assert.throws(() => partitionForAgent('../escape', KEY));
  assert.throws(() => partitionForAgent('agent_A', Buffer.alloc(8)));
});

test('tabs of different agents use isolated partitions and hardened web preferences', () => {
  const { manager, sessions, views } = setup();
  manager.openTab('agent_A', 'https://example.com');
  manager.openTab('agent_A', 'https://example.org');
  manager.openTab('agent_B', 'https://example.com');

  const [a1, a2, b1] = views.map((view) => view.options.webPreferences);
  assert.equal(a1.partition, manager.partitionFor('agent_A'));
  assert.equal(a2.partition, a1.partition);
  assert.equal(b1.partition, manager.partitionFor('agent_B'));
  assert.notEqual(a1.partition, b1.partition);
  assert.equal(sessions.size, 2);
  for (const prefs of [a1, a2, b1]) {
    assert.equal(prefs.sandbox, true);
    assert.equal(prefs.contextIsolation, true);
    assert.equal(prefs.nodeIntegration, false);
    assert.equal(prefs.nodeIntegrationInSubFrames, false);
    assert.equal(prefs.webviewTag, false);
    assert.equal(prefs.webSecurity, true);
    assert.equal(prefs.preload, undefined);
  }
  assert.deepEqual(
    manager.listTabs('agent_A').map((tab) => tab.agentId),
    ['agent_A', 'agent_A'],
  );
  assert.equal(manager.listTabs('agent_B').length, 1);
  assert.throws(() => manager.openTab('bad id!'));
});

test('every permission and device request is denied by default', async () => {
  const { manager, sessions } = setup();
  manager.openTab('agent_A');
  const session = sessions.get(manager.partitionFor('agent_A'));

  for (const permission of ['media', 'geolocation', 'notifications', 'clipboard-read', 'midi', 'display-capture']) {
    const granted = await new Promise((resolve) => session.permissionRequest(null, permission, resolve, {}));
    assert.equal(granted, false, permission);
    assert.equal(session.permissionCheck(null, permission, 'https://example.com', {}), false, permission);
  }
  assert.equal(session.devicePermission({ deviceType: 'usb' }), false);
  session.bluetoothPairing({}, (response) => assert.deepEqual(response, { confirmed: false }));

  for (const [event, args] of [
    ['select-hid-device', [{}]],
    ['select-serial-port', [[], {}]],
    ['select-usb-device', [{}]],
  ]) {
    const ev = navigationEvent('');
    let selected = 'not-called';
    session.emit(event, ev, ...args, (id) => {
      selected = id;
    });
    assert.equal(ev.defaultPrevented, true, event);
    assert.ok(selected === null || selected === '' || selected === undefined, event);
  }
});

test('downloads are cancelled', () => {
  const { manager, sessions } = setup();
  manager.openTab('agent_A');
  const ev = navigationEvent('');
  sessions.get(manager.partitionFor('agent_A')).emit('will-download', ev, { getURL: () => 'https://x.test/f.zip' });
  assert.equal(ev.defaultPrevented, true);
});

test('only http and https can be opened or navigated to', () => {
  for (const url of ['https://example.com/a?b#c', 'http://localhost:5173/']) {
    assert.equal(isNavigableUrl(url), true, url);
  }
  for (const url of [
    'file:///etc/passwd',
    'javascript:alert(1)',
    'data:text/html,hi',
    'chrome://settings',
    'devtools://devtools',
    'app://calango/',
    'mailto:a@b.c',
    'ftp://example.com',
    'https://user:pass@example.com',
    '',
    42,
  ]) {
    assert.equal(isNavigableUrl(url), false, String(url));
  }
  assert.equal(normalizeAddress('example.com'), 'https://example.com/');
  assert.equal(normalizeAddress('localhost:5173/app'), 'http://localhost:5173/app');
  assert.equal(normalizeAddress('127.0.0.1:8080'), 'http://127.0.0.1:8080/');
  assert.equal(normalizeAddress('http://example.com'), 'http://example.com/');
  assert.equal(normalizeAddress('javascript:alert(1)'), null);
  assert.equal(normalizeAddress('file:///etc/passwd'), null);
  assert.equal(normalizeAddress('mailto:a@b.c'), null);
  assert.equal(normalizeAddress('two words'), null);

  const { manager, views } = setup();
  assert.throws(() => manager.openTab('agent_A', 'file:///etc/passwd'));
  const tab = manager.openTab('agent_A', 'https://example.com');
  assert.throws(() => manager.navigate(tab.id, 'javascript:alert(1)'));
  assert.deepEqual(views[0].webContents.loaded, ['https://example.com/']);
});

test('page-initiated navigations and redirects outside http(s) are cancelled', () => {
  const { manager, views } = setup();
  manager.openTab('agent_A', 'https://example.com');
  const contents = views[0].webContents;

  const blocked = navigationEvent('file:///etc/passwd');
  contents.emit('will-navigate', blocked);
  assert.equal(blocked.defaultPrevented, true);

  const allowed = navigationEvent('https://example.org/');
  contents.emit('will-navigate', allowed);
  assert.equal(allowed.defaultPrevented, false);

  const redirect = navigationEvent('calango-evil://x');
  contents.emit('will-redirect', redirect);
  assert.equal(redirect.defaultPrevented, true);

  const webview = navigationEvent('');
  contents.emit('will-attach-webview', webview);
  assert.equal(webview.defaultPrevented, true);
});

test('window.open becomes a tab of the same agent, never a free window', () => {
  assert.equal(decideWindowOpen('https://example.com'), 'tab');
  assert.equal(decideWindowOpen('file:///etc/passwd'), 'deny');

  const { manager, views } = setup();
  manager.openTab('agent_A', 'https://example.com');
  const contents = views[0].webContents;

  assert.deepEqual(contents.windowOpen({ url: 'https://popup.example/', disposition: 'new-window' }), {
    action: 'deny',
  });
  assert.equal(views.length, 2);
  assert.equal(views[1].options.webPreferences.partition, manager.partitionFor('agent_A'));
  assert.deepEqual(views[1].webContents.loaded, ['https://popup.example/']);
  assert.equal(manager.listTabs('agent_A').length, 2);

  assert.deepEqual(contents.windowOpen({ url: 'javascript:alert(1)', disposition: 'foreground-tab' }), {
    action: 'deny',
  });
  assert.equal(views.length, 2);
});

test('clearing an agent profile closes its tabs and erases only its partition', async () => {
  const { manager, sessions, views } = setup();
  manager.openTab('agent_A', 'https://example.com');
  manager.openTab('agent_B', 'https://example.com');

  await manager.clearProfile('agent_A');

  assert.equal(views[0].webContents.destroyed, true);
  assert.equal(views[1].webContents.destroyed, false);
  assert.deepEqual(manager.listTabs('agent_A'), []);
  assert.equal(manager.listTabs('agent_B').length, 1);
  assert.deepEqual(sessions.get(manager.partitionFor('agent_A')).cleared, ['data', 'storage', 'auth']);
  assert.deepEqual(sessions.get(manager.partitionFor('agent_B')).cleared, []);
});

test('only the active tab of the focused agent is shown', () => {
  const { manager, views, attached } = setup();
  manager.focusAgent('agent_A');
  manager.setBounds({ x: 10, y: 20, width: 300, height: 200 });
  const first = manager.openTab('agent_A', 'https://a.example');
  assert.deepEqual([...attached], [views[0]]);
  assert.deepEqual(views[0].bounds, { x: 10, y: 20, width: 300, height: 200 });

  const second = manager.openTab('agent_A', 'https://b.example');
  assert.deepEqual([...attached], [views[1]]);

  // A popup of a background agent opens a tab but does not take over the panel.
  manager.openTab('agent_B', 'https://c.example');
  assert.deepEqual([...attached], [views[1]]);

  manager.activate(first.id);
  assert.deepEqual([...attached], [views[0]]);
  manager.closeTab(first.id);
  assert.deepEqual([...attached], [views[1]]);
  assert.equal(manager.listTabs('agent_A').find((tab) => tab.id === second.id).active, true);

  manager.setBounds(null);
  assert.equal(attached.size, 0);
  manager.focusAgent('agent_B');
  manager.setBounds({ x: 0, y: 0, width: 100, height: 100 });
  assert.deepEqual([...attached], [views[2]]);
});

test('renderer bounds are validated and clamped to the window', () => {
  const max = { width: 800, height: 600 };
  assert.deepEqual(parseBounds({ x: 10.4, y: -5, width: 2000, height: 100 }, max), {
    x: 10,
    y: 0,
    width: 790,
    height: 100,
  });
  assert.equal(parseBounds({ x: 'a', y: 0, width: 1, height: 1 }, max), null);
  assert.equal(parseBounds({ x: Number.NaN, y: 0, width: 1, height: 1 }, max), null);
  assert.equal(parseBounds(null, max), null);
});

test('profile key is created once with owner-only permissions and reused', async (context) => {
  const dir = await mkdtemp(path.join(os.tmpdir(), 'calango-browser-'));
  context.after(() => rm(dir, { recursive: true, force: true }));
  const file = path.join(dir, 'nested', 'browser-profile.key');
  const key = await loadProfileKey(file);
  assert.equal(key.byteLength, 32);
  assert.deepEqual(await loadProfileKey(file), key);
  assert.deepEqual(await readFile(file), key);
  if (process.platform !== 'win32') {
    assert.equal((await stat(file)).mode & 0o777, 0o600);
  }
});

test('dispose detaches views and closes every tab without waiting for beforeunload', () => {
  const { manager, views, attached } = setup();
  manager.focusAgent('agent_A');
  manager.setBounds({ x: 0, y: 0, width: 100, height: 100 });
  manager.openTab('agent_A', 'https://a.example');
  manager.openTab('agent_B', 'https://b.example');
  assert.equal(attached.size, 1);

  manager.dispose();

  assert.equal(attached.size, 0);
  for (const view of views) {
    assert.equal(view.webContents.destroyed, true);
    assert.deepEqual(view.webContents.closeOptions, { waitForBeforeUnload: false });
  }
  assert.deepEqual(manager.listTabs('agent_A'), []);
  manager.dispose();
});

test('dispose still closes tabs when the window is already destroyed (quit)', () => {
  let windowDestroyed = false;
  const { manager, views } = setup({
    remove: () => {
      if (windowDestroyed) throw new Error('Object has been destroyed');
    },
  });
  manager.focusAgent('agent_A');
  manager.setBounds({ x: 0, y: 0, width: 100, height: 100 });
  manager.openTab('agent_A', 'https://a.example');
  manager.openTab('agent_A', 'https://b.example');

  windowDestroyed = true;
  assert.doesNotThrow(() => manager.dispose());
  assert.ok(views.every((view) => view.webContents.destroyed));
});
