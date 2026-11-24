const assert = require('node:assert/strict');
const { mkdtemp, readFile, rm } = require('node:fs/promises');
const { test } = require('node:test');
const os = require('node:os');
const path = require('node:path');
const {
  linuxAutostartEnabled,
  linuxAutostartEntry,
  linuxAutostartFilePath,
  setLinuxAutostart,
} = require('../dist/main/startup.js');

test('Linux autostart honors XDG_CONFIG_HOME and falls back for relative values', () => {
  assert.equal(
    linuxAutostartFilePath('/tmp/calango-config', '/home/tester'),
    '/tmp/calango-config/autostart/calango.desktop',
  );
  assert.equal(
    linuxAutostartFilePath('relative-config', '/home/tester'),
    '/home/tester/.config/autostart/calango.desktop',
  );
});

test('Linux autostart writes a quoted entry and removes it when disabled', async (context) => {
  const home = await mkdtemp(path.join(os.tmpdir(), 'calango-autostart-'));
  context.after(() => rm(home, { recursive: true, force: true }));
  const filePath = linuxAutostartFilePath(path.join(home, 'config'), home);
  const executable = path.join(home, 'Electron App');
  const appPath = path.join(home, 'Calango 100% App');
  const entry = linuxAutostartEntry(executable, [appPath]);

  assert.match(entry, /Exec=".*Electron App" ".*Calango 100%% App"/);
  await setLinuxAutostart(filePath, true, executable, [appPath]);
  assert.equal(await linuxAutostartEnabled(filePath), true);
  assert.equal(await readFile(filePath, 'utf8'), entry);

  await setLinuxAutostart(filePath, false, executable, [appPath]);
  assert.equal(await linuxAutostartEnabled(filePath), false);
});
