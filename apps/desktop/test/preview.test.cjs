const assert = require('node:assert/strict');
const { EventEmitter } = require('node:events');
const http = require('node:http');
const net = require('node:net');
const { test } = require('node:test');
const {
  DEFAULT_PREVIEW_PORTS,
  PREVIEW_OWNER,
  PREVIEW_PARTITION_PREFIX,
  cleanTitle,
  detectDevServers,
  isWithinPreviewOrigin,
  parsePortList,
  parsePreviewTarget,
  partitionForPreview,
  probePort,
} = require('../dist/main/browser/preview.js');
const { AGENT_PARTITION_PREFIX } = require('../dist/main/browser/policy.js');
const { BrowserManager } = require('../dist/main/browser/manager.js');

const KEY = Buffer.alloc(32, 7);

// Electron fakes: only what the manager and the hardening touch.
class FakeSession extends EventEmitter {
  constructor(partition) {
    super();
    this.partition = partition;
    this.cleared = [];
    this.beforeRequest = null;
    this.webRequest = {
      onBeforeRequest: (listener) => {
        this.beforeRequest = listener;
      },
    };
  }
  setPermissionRequestHandler(handler) {
    this.permissionRequest = handler;
  }
  setPermissionCheckHandler(handler) {
    this.permissionCheck = handler;
  }
  setDevicePermissionHandler() {}
  setBluetoothPairingHandler() {}
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

let nextContentsId = 1;
class FakeWebContents extends EventEmitter {
  constructor() {
    super();
    this.id = nextContentsId++;
    this.loaded = [];
    this.destroyed = false;
    this.url = '';
    this.navigationHistory = { canGoBack: () => false, canGoForward: () => false };
  }
  setWindowOpenHandler(handler) {
    this.windowOpen = handler;
  }
  loadURL(url) {
    this.loaded.push(url);
    this.url = url;
    return Promise.resolve();
  }
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
  close() {
    this.destroyed = true;
  }
}

function setup(extra = {}) {
  const sessions = new Map();
  const views = [];
  const blocked = [];
  const external = [];
  const manager = new BrowserManager({
    profileKey: KEY,
    fromPartition: (partition) => {
      if (!sessions.has(partition)) sessions.set(partition, new FakeSession(partition));
      return sessions.get(partition);
    },
    createView: (options) => {
      const view = { options, webContents: new FakeWebContents(), setBounds() {} };
      views.push(view);
      return view;
    },
    host: { add() {}, remove() {} },
    onTabsChanged() {},
    onPreviewBlocked: (event) => blocked.push(event),
    openExternal: (url) => {
      external.push(url);
    },
    reservedPorts: [7317],
    ...extra,
  });
  return { manager, sessions, views, blocked, external };
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

function beforeRequest(session, details) {
  let response;
  session.beforeRequest(details, (value) => {
    response = value;
  });
  return response;
}

test('preview targets accept only plain http on loopback', () => {
  assert.deepEqual(parsePreviewTarget(5173), {
    origin: 'http://127.0.0.1:5173',
    url: 'http://127.0.0.1:5173/',
    port: 5173,
  });
  assert.equal(parsePreviewTarget('5173').origin, 'http://127.0.0.1:5173');
  assert.equal(parsePreviewTarget('localhost:3000').origin, 'http://localhost:3000');
  assert.equal(parsePreviewTarget('LOCALHOST:3000/app?x=1').url, 'http://localhost:3000/app?x=1');
  assert.equal(parsePreviewTarget('http://127.0.0.1:8080/a#b').url, 'http://127.0.0.1:8080/a#b');
  assert.equal(parsePreviewTarget('http://[::1]:4200').origin, 'http://[::1]:4200');
  assert.equal(parsePreviewTarget('http://localhost').origin, 'http://localhost:80');
  // Numeric forms of 127.0.0.1 are normalized by the URL parser and stay loopback.
  assert.equal(parsePreviewTarget('http://0x7f.1:3000').origin, 'http://127.0.0.1:3000');

  for (const bad of [
    0,
    65536,
    -1,
    3000.5,
    '0',
    '',
    '   ',
    'http://0.0.0.0:3000',
    '0.0.0.0:3000',
    'http://[::]:3000',
    'http://192.168.0.10:3000',
    'http://10.0.0.1:3000',
    'http://8.8.8.8:80',
    'http://127.0.0.2:3000',
    'http://[::ffff:127.0.0.1]:3000',
    'http://localhost.evil.com:3000',
    'localhost.evil.com:3000',
    'http://localhost.:3000',
    'http://app.localhost:3000',
    'http://localtest.me:3000',
    'http://evil.com#@localhost:3000',
    'http://user@localhost:3000',
    'http://user:pass@127.0.0.1:3000',
    'https://localhost:3000',
    'ws://localhost:3000',
    'file:///etc/passwd',
    'javascript:alert(1)',
    'localhost:0',
    'two words',
    null,
    {},
  ]) {
    assert.equal(parsePreviewTarget(bad), null, String(bad));
  }
});

test('preview containment keeps the tab on the chosen loopback port', () => {
  const origin = 'http://127.0.0.1:5173';
  for (const url of [
    'http://127.0.0.1:5173/',
    'http://127.0.0.1:5173/deep/path?q=1#h',
    'http://localhost:5173/',
    'http://[::1]:5173/x',
  ]) {
    assert.equal(isWithinPreviewOrigin(url, origin), true, url);
  }
  for (const url of [
    'http://127.0.0.1:5174/',
    'http://127.0.0.1:7317/v1/agents',
    'https://127.0.0.1:5173/',
    'http://localhost.evil.com:5173/',
    'http://user@localhost:5173/',
    'http://example.com/',
    'https://example.com:5173/',
    'http://0.0.0.0:5173/',
    'file:///etc/passwd',
    'javascript:alert(1)',
    '',
    undefined,
  ]) {
    assert.equal(isWithinPreviewOrigin(url, origin), false, String(url));
  }
});

test('preview partition is separate from agent partitions and hookable per project', () => {
  const shared = partitionForPreview(null, KEY);
  assert.match(shared, /^persist:calango-preview-[0-9a-f]{32}$/);
  assert.ok(shared.startsWith(PREVIEW_PARTITION_PREFIX));
  assert.ok(!shared.startsWith(AGENT_PARTITION_PREFIX));
  assert.equal(shared, partitionForPreview(null, KEY));
  assert.notEqual(partitionForPreview('proj_1', KEY), shared);
  assert.notEqual(partitionForPreview('proj_1', KEY), partitionForPreview('proj_2', KEY));
  assert.throws(() => partitionForPreview('../x', KEY));
  assert.throws(() => partitionForPreview(null, Buffer.alloc(4)));
});

test('port lists are short and validated', () => {
  assert.deepEqual(parsePortList(undefined), [...DEFAULT_PREVIEW_PORTS]);
  assert.ok(DEFAULT_PREVIEW_PORTS.length <= 12);
  assert.deepEqual(parsePortList([3000, '3000', 5173]), [3000, 5173]);
  assert.equal(parsePortList([]), null);
  assert.equal(parsePortList([1, 2, 'x']), null);
  assert.equal(parsePortList(Array.from({ length: 33 }, (_v, i) => 3000 + i)), null);
  assert.equal(parsePortList('3000'), null);
});

test('page titles become short plain text', () => {
  assert.equal(cleanTitle('  My &amp; App\n\t<b>  '), 'My & App <b>');
  assert.equal(cleanTitle('a‮b\u0000c'), 'a b c');
  assert.equal(cleanTitle('x'.repeat(500)).length, 120);
});

test('preview tabs use the preview partition, never an agent one', () => {
  const { manager, views, sessions } = setup();
  const tab = manager.openPreview(5173);
  assert.equal(tab.agentId, PREVIEW_OWNER);
  assert.deepEqual(tab.preview, { origin: 'http://127.0.0.1:5173', autoReload: false });
  const prefs = views[0].options.webPreferences;
  assert.equal(prefs.partition, manager.previewPartitionFor(null));
  assert.notEqual(prefs.partition, manager.partitionFor('agent_A'));
  assert.equal(prefs.sandbox, true);
  assert.equal(prefs.nodeIntegration, false);
  assert.equal(prefs.preload, undefined);
  assert.deepEqual(views[0].webContents.loaded, ['http://127.0.0.1:5173/']);
  assert.deepEqual(
    manager.listTabs(PREVIEW_OWNER).map((t) => t.id),
    [tab.id],
  );

  // Agent tabs do not see the preview, and an agent id cannot be the preview owner.
  manager.openTab('agent_A', 'https://example.com');
  assert.equal(manager.listTabs('agent_A').length, 1);
  assert.equal(manager.listTabs(PREVIEW_OWNER).length, 1);
  assert.throws(() => manager.openTab(PREVIEW_OWNER, 'https://example.com'));

  // Permissions and downloads are denied like in agent tabs.
  const session = sessions.get(manager.previewPartitionFor(null));
  session.permissionRequest(null, 'media', (granted) => assert.equal(granted, false), {});
  assert.equal(session.permissionCheck(null, 'geolocation', 'http://127.0.0.1:5173', {}), false);
  const download = navigationEvent('');
  session.emit('will-download', download, { getURL: () => 'http://127.0.0.1:5173/f.zip' });
  assert.equal(download.defaultPrevented, true);

  // A project id (hook for #82) gets its own partition.
  manager.openPreview(3000, { projectId: 'proj_1' });
  assert.equal(views[2].options.webPreferences.partition, manager.previewPartitionFor('proj_1'));
  assert.throws(() => manager.openPreview(3000, { projectId: 'bad id' }));
});

test('preview refuses non-loopback targets and the reserved daemon port', () => {
  const { manager, views } = setup();
  for (const bad of [
    'http://0.0.0.0:3000',
    'http://192.168.1.2:3000',
    'localhost.evil.com:3000',
    'https://localhost:3000',
  ]) {
    assert.throws(() => manager.openPreview(bad), /loopback/, bad);
  }
  assert.throws(() => manager.openPreview(7317), /reserved/);
  assert.equal(views.length, 0);
});

test('preview tab navigations, redirects and popups stay inside the origin', () => {
  const { manager, views, blocked, external } = setup();
  const tab = manager.openPreview('localhost:5173');
  const contents = views[0].webContents;

  const inside = navigationEvent('http://localhost:5173/about');
  contents.emit('will-navigate', inside);
  assert.equal(inside.defaultPrevented, false);
  const alias = navigationEvent('http://127.0.0.1:5173/x');
  contents.emit('will-navigate', alias);
  assert.equal(alias.defaultPrevented, false);

  const outside = navigationEvent('https://example.com/');
  contents.emit('will-navigate', outside);
  assert.equal(outside.defaultPrevented, true);
  const otherPort = navigationEvent('http://localhost:7317/v1/agents');
  contents.emit('will-navigate', otherPort);
  assert.equal(otherPort.defaultPrevented, true);
  const fileNav = navigationEvent('file:///etc/passwd');
  contents.emit('will-navigate', fileNav);
  assert.equal(fileNav.defaultPrevented, true);

  const redirect = navigationEvent('https://evil.example/');
  contents.emit('will-redirect', redirect);
  assert.equal(redirect.defaultPrevented, true);
  const subframeRedirect = navigationEvent('https://cdn.example/frame', false);
  contents.emit('will-redirect', subframeRedirect);
  assert.equal(subframeRedirect.defaultPrevented, false);

  assert.deepEqual(contents.windowOpen({ url: 'https://popup.example/' }), { action: 'deny' });
  assert.equal(views.length, 1);
  assert.deepEqual(contents.windowOpen({ url: 'http://localhost:5173/popup' }), { action: 'deny' });
  assert.equal(views.length, 2);
  assert.equal(views[1].options.webPreferences.partition, manager.previewPartitionFor(null));
  assert.deepEqual(views[1].webContents.loaded, ['http://localhost:5173/popup']);
  assert.equal(manager.listTabs(PREVIEW_OWNER)[1].preview.origin, 'http://localhost:5173');

  // Only http(s) URLs that left the origin are reported (not file:).
  assert.deepEqual(
    blocked.map((event) => event.url),
    ['https://example.com/', 'http://localhost:7317/v1/agents', 'https://evil.example/', 'https://popup.example/'],
  );
  assert.ok(blocked.every((event) => event.tabId === tab.id));
  assert.deepEqual(external, []);

  // The address bar of a preview tab cannot leave the origin either.
  assert.throws(() => manager.navigate(tab.id, 'https://example.com'), /local address/);
  manager.navigate(tab.id, 'http://localhost:5173/settings');
  assert.equal(contents.loaded.at(-1), 'http://localhost:5173/settings');
});

test('the network layer cancels top-level loads outside the preview origin', () => {
  const { manager, views, sessions } = setup();
  manager.openPreview(5173);
  const session = sessions.get(manager.previewPartitionFor(null));
  const id = views[0].webContents.id;

  const main = (url, webContentsId = id) => beforeRequest(session, { url, resourceType: 'mainFrame', webContentsId });
  assert.deepEqual(main('http://127.0.0.1:5173/page'), {});
  assert.deepEqual(main('https://example.com/'), { cancel: true });
  assert.deepEqual(main('http://127.0.0.1:5173/', 9999), { cancel: true });
  // Subresources are not filtered.
  assert.deepEqual(
    beforeRequest(session, { url: 'https://cdn.example/a.js', resourceType: 'script', webContentsId: id }),
    {},
  );

  // Agent partitions get no such filter.
  manager.openTab('agent_A', 'https://example.com');
  assert.equal(sessions.get(manager.partitionFor('agent_A')).beforeRequest, null);
});

test('only a URL a preview was kept from can go to the system browser, once', async () => {
  const { manager, views, external } = setup();
  manager.openPreview(5173);
  views[0].webContents.emit('will-navigate', navigationEvent('https://docs.example/'));

  assert.equal(await manager.openExternalBlocked('https://other.example/'), false);
  assert.equal(await manager.openExternalBlocked('file:///etc/passwd'), false);
  assert.equal(await manager.openExternalBlocked(42), false);
  assert.equal(await manager.openExternalBlocked('https://docs.example/'), true);
  assert.equal(await manager.openExternalBlocked('https://docs.example/'), false);
  assert.deepEqual(external, ['https://docs.example/']);
});

test('auto reload retries when the dev server is down, only if enabled', async (context) => {
  context.mock.timers.enable({ apis: ['setTimeout'] });
  const { manager, views } = setup();
  manager.openPreview(5173, { autoReload: true });
  manager.openPreview(5174);
  const [on, off] = views.map((view) => view.webContents);

  on.emit('did-fail-load', {}, -102, 'ERR_CONNECTION_REFUSED', 'http://127.0.0.1:5173/app', true);
  off.emit('did-fail-load', {}, -102, 'ERR_CONNECTION_REFUSED', 'http://127.0.0.1:5174/', true);
  // Not retried: subframe failures, aborted or blocked loads.
  on.emit('did-fail-load', {}, -102, '', 'http://127.0.0.1:5173/', false);
  context.mock.timers.tick(2_000);
  assert.deepEqual(on.loaded, ['http://127.0.0.1:5173/', 'http://127.0.0.1:5173/app']);
  assert.deepEqual(off.loaded, ['http://127.0.0.1:5174/']);

  on.emit('did-fail-load', {}, -20, 'ERR_BLOCKED_BY_CLIENT', 'https://example.com/', true);
  on.emit('did-fail-load', {}, -3, 'ERR_ABORTED', 'http://127.0.0.1:5173/', true);
  context.mock.timers.tick(20_000);
  assert.equal(on.loaded.length, 2);

  // Backoff, and closing the tab cancels the pending retry.
  on.emit('did-fail-load', {}, -102, '', 'http://127.0.0.1:5173/app', true);
  context.mock.timers.tick(3_999);
  assert.equal(on.loaded.length, 2);
  context.mock.timers.tick(1);
  assert.equal(on.loaded.length, 3);
  on.emit('did-fail-load', {}, -102, '', 'http://127.0.0.1:5173/app', true);
  manager.closeTab(manager.listTabs(PREVIEW_OWNER)[0].id);
  context.mock.timers.tick(60_000);
  assert.equal(on.loaded.length, 3);
});

test('clearing the preview profile closes preview tabs only', async () => {
  const { manager, views, sessions } = setup();
  manager.openPreview(5173);
  manager.openTab('agent_A', 'https://example.com');
  await manager.clearProfile(PREVIEW_OWNER);
  assert.equal(views[0].webContents.destroyed, true);
  assert.equal(views[1].webContents.destroyed, false);
  assert.deepEqual(sessions.get(manager.previewPartitionFor(null)).cleared, ['data', 'storage', 'auth']);
  assert.deepEqual(sessions.get(manager.partitionFor('agent_A')).cleared, []);
});

// --- Loopback probe against real sockets ---

function listen(server, host = '127.0.0.1') {
  return new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, host, () => resolve(server.address().port));
  });
}

test('probe finds a local HTTP server, its title and kind', async (context) => {
  const seen = [];
  const server = http.createServer((req, res) => {
    seen.push(req.headers.host);
    res.setHeader('Content-Type', 'text/html');
    res.end(
      '<!doctype html><html><head><title>Meu &amp; App</title><script type="module" src="/@vite/client"></script></head></html>',
    );
  });
  const port = await listen(server);
  context.after(() => server.close());

  const found = await probePort(port);
  assert.deepEqual(found, { port, origin: `http://127.0.0.1:${port}`, status: 200, title: 'Meu & App', kind: 'vite' });
  assert.deepEqual(seen, [`localhost:${port}`]);

  const list = await detectDevServers([port, port]);
  assert.equal(list.length, 1);
  assert.deepEqual(await detectDevServers([port], { exclude: [port] }), []);
});

test('probe reports a server that answers with an error status or no title', async (context) => {
  const server = http.createServer((_req, res) => {
    res.statusCode = 404;
    res.setHeader('Server', 'SimpleHTTP/0.6 Python/3.12');
    res.end('not found');
  });
  const port = await listen(server);
  context.after(() => server.close());
  assert.deepEqual(await probePort(port), {
    port,
    origin: `http://127.0.0.1:${port}`,
    status: 404,
    title: '',
    kind: 'python',
  });
});

test('probe returns null for a closed port', async () => {
  const server = http.createServer();
  const port = await listen(server);
  await new Promise((resolve) => server.close(resolve));
  const started = Date.now();
  assert.equal(await probePort(port, { timeoutMs: 300 }), null);
  assert.ok(Date.now() - started < 2_000);
  assert.equal(await probePort(0), null);
  assert.equal(await probePort(70000), null);
});

test('probe gives up on a server that never answers', async (context) => {
  const sockets = [];
  const server = net.createServer((socket) => sockets.push(socket));
  const port = await listen(server);
  context.after(() => {
    for (const socket of sockets) socket.destroy();
    server.close();
  });
  const started = Date.now();
  assert.equal(await probePort(port, { timeoutMs: 200 }), null);
  const elapsed = Date.now() - started;
  // 127.0.0.1 times out; ::1 is refused at once (or times out too when nothing binds there).
  assert.ok(elapsed >= 190 && elapsed < 1_500, `elapsed ${elapsed}`);
});

test('probe falls back to ::1 when the server listens on IPv6 loopback only', async (context) => {
  const server = http.createServer((_req, res) => res.end('<title>v6</title>'));
  let port;
  try {
    port = await listen(server, '::1');
  } catch {
    context.skip('no IPv6 loopback');
    return;
  }
  context.after(() => server.close());
  const found = await probePort(port);
  assert.equal(found?.origin, `http://[::1]:${port}`);
  assert.equal(found?.title, 'v6');
});
