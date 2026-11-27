import assert from 'node:assert/strict';
import { chmod, cp, mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';
import { expectedLegalFiles, requiredLegalFiles, verifyPackage } from './verify-desktop-package.mjs';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

// fakePackage monta uma pasta resources como o electron-builder monta, a partir do repositório.
async function fakePackage(context) {
  const dir = await mkdtemp(path.join(os.tmpdir(), 'calango-pkg-'));
  context.after(() => rm(dir, { recursive: true, force: true }));
  const resources = path.join(dir, 'resources');
  await mkdir(path.join(resources, 'legal'), { recursive: true });
  await cp(path.join(repoRoot, 'LICENSE'), path.join(resources, 'legal', 'LICENSE'));
  await cp(path.join(repoRoot, 'NOTICE'), path.join(resources, 'legal', 'NOTICE'));
  await cp(path.join(repoRoot, 'third_party'), path.join(resources, 'legal', 'third_party'), { recursive: true });
  await mkdir(path.join(resources, 'bin'));
  await writeFile(path.join(resources, 'bin', 'calango'), '#!/bin/sh\n');
  await chmod(path.join(resources, 'bin', 'calango'), 0o755);
  return resources;
}

test('the repository has every legal file the packages must carry', async () => {
  const files = await expectedLegalFiles();
  for (const f of requiredLegalFiles) assert.ok(files.includes(f), f);
});

test('a complete package passes', async (context) => {
  const resources = await fakePackage(context);
  assert.deepEqual(await verifyPackage({ resources, platform: 'linux' }), []);
});

test('a package without the IAPro-SDK NOTICE fails', async (context) => {
  const resources = await fakePackage(context);
  await rm(path.join(resources, 'legal', 'third_party', 'iapro-sdk', 'NOTICE'));
  assert.deepEqual(await verifyPackage({ resources, platform: 'linux' }), [
    'ausente no pacote: legal/third_party/iapro-sdk/NOTICE',
  ]);
});

test('a package with a changed LICENSE or a missing daemon fails', async (context) => {
  const resources = await fakePackage(context);
  await writeFile(path.join(resources, 'legal', 'LICENSE'), 'outra licença\n');
  await rm(path.join(resources, 'bin', 'calango'));
  assert.deepEqual(await verifyPackage({ resources, platform: 'linux' }), [
    'diferente do repositório: legal/LICENSE',
    'ausente no pacote: bin/calango',
  ]);
});

test('a daemon without the executable bit fails outside Windows', async (context) => {
  const resources = await fakePackage(context);
  await chmod(path.join(resources, 'bin', 'calango'), 0o644);
  assert.deepEqual(await verifyPackage({ resources, platform: 'linux' }), [
    'daemon sem permissão de execução: bin/calango',
  ]);
});

// Leitura textual de apps/desktop/electron-builder.yml (sem dependência de parser YAML).
function section(text, name) {
  const match = new RegExp(`^${name}:\\n((?:[ #].*\\n|\\n)*)`, 'm').exec(text);
  assert.ok(match, `seção ${name} ausente no electron-builder.yml`);
  return match[1];
}

test('electron-builder copies LICENSE, NOTICE, third_party and the daemon for every target', async () => {
  const config = await readFile(path.join(repoRoot, 'apps/desktop/electron-builder.yml'), 'utf8');
  const shared = section(config, 'extraResources');
  for (const [from, to] of [
    ['../../LICENSE', 'legal/LICENSE'],
    ['../../NOTICE', 'legal/NOTICE'],
    ['../../third_party', 'legal/third_party'],
  ]) {
    assert.match(shared, new RegExp(`- from: ${from.replaceAll('.', '\\.')}\\n\\s+to: ${to}\\n`));
  }
  assert.match(section(config, 'win'), /to: bin\/calango\.exe\n/);
  const linux = section(config, 'linux');
  assert.match(linux, /to: bin\/calango\n/);
  assert.match(linux, /- target: deb\n/);
  assert.match(linux, /- target: AppImage\n/);
  const mac = section(config, 'mac');
  assert.match(mac, /to: bin\/calango\n/);
  assert.match(mac, /- target: dmg\n/);
  // macOS fica configurado, mas sem assinatura nem notarização até haver credenciais (#40).
  assert.match(mac, /^ {2}identity: null$/m);
  assert.match(mac, /^ {2}notarize: false$/m);
  assert.match(config, /^publish: null$/m);
});
