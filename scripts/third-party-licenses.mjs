#!/usr/bin/env node
// Keeps the license texts of third-party code that ships inside Calango under
// third_party/, and checks that NOTICE names each one with its version (issue #50).
//
//   node scripts/third-party-licenses.mjs           check (CI): fails on any drift
//   node scripts/third-party-licenses.mjs --write   copy the texts again
//
// Covered: the Go toolchain and every Go module linked into cmd/calango for linux,
// windows and darwin, and the npm packages bundled into the desktop renderer. NOTICE is
// written by hand; the script only reports what is missing there. Needs Go and the
// modules in the module cache (`go mod download`).
import { execFileSync } from 'node:child_process';
import { existsSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const outRoot = path.join(root, 'third_party');
const write = process.argv.includes('--write');

// Packages Vite bundles into dist/renderer. Minification drops their license comments,
// so their texts must ship here. Update when the renderer imports a new runtime package.
const rendererPackages = ['react', 'react-dom', 'scheduler'];
// Modules whose texts already live in a hand-maintained third_party folder.
const handled = { 'github.com/IAPro-Community/IAPro-SDK': 'iapro-sdk' };
const licenseFile = /^(licen[cs]e|copying|copyright|notice|patents)([-_][a-z0-9-]+)?(\.(txt|md))?$/i;

function go(args, env = {}) {
  return execFileSync('go', args, { cwd: root, encoding: 'utf8', env: { ...process.env, ...env } });
}

function licenseTexts(dir, prefix = '') {
  return readdirSync(dir)
    .filter((name) => licenseFile.test(name))
    .sort()
    .map((name) => ({ name: prefix + name, text: readFileSync(path.join(dir, name), 'utf8') }));
}

// moduleTexts adds the license files of linked subpackages, such as vendored code with
// its own license, under their path inside the module.
function moduleTexts(m) {
  const texts = licenseTexts(m.dir);
  for (const pkgDir of [...m.pkgDirs].sort()) {
    if (pkgDir !== m.dir) {
      texts.push(...licenseTexts(pkgDir, `${path.relative(m.dir, pkgDir).split(path.sep).join('/')}/`));
    }
  }
  return texts;
}

function goModules() {
  const modules = new Map();
  for (const goos of ['linux', 'windows', 'darwin']) {
    const out = go(
      [
        'list',
        '-deps',
        '-f',
        '{{if .Module}}{{if not .Module.Main}}{{.Module.Path}}\t{{.Module.Version}}\t{{.Module.Dir}}\t{{.Dir}}{{end}}{{end}}',
        './cmd/calango',
      ],
      { GOOS: goos, GOARCH: 'amd64', CGO_ENABLED: '0' },
    );
    for (const line of out.split('\n').filter(Boolean)) {
      const [modPath, version, dir, pkgDir] = line.split('\t');
      const entry = modules.get(modPath) ?? { modPath, version, dir, pkgDirs: new Set() };
      entry.pkgDirs.add(pkgDir);
      modules.set(modPath, entry);
    }
  }
  return [...modules.values()].sort((a, b) => a.modPath.localeCompare(b.modPath));
}

function npmPackage(name) {
  const dir = path.join(root, 'node_modules', name);
  const { version } = JSON.parse(readFileSync(path.join(dir, 'package.json'), 'utf8'));
  return { name, version, dir };
}

// Each target: the folder under third_party/, the texts it must hold, and the NOTICE line.
const targets = [];
const goroot = go(['env', 'GOROOT']).trim();
targets.push({ folder: 'go/go-toolchain', texts: licenseTexts(goroot), notice: 'Go toolchain' });
for (const m of goModules()) {
  const texts = moduleTexts(m);
  if (texts.length === 0) {
    throw new Error(`${m.modPath} ${m.version} has no license file in ${m.dir}`);
  }
  targets.push({
    folder: handled[m.modPath] ?? `go/${m.modPath}`,
    texts: handled[m.modPath] ? texts.filter((t) => /^licen[cs]e$/i.test(t.name)) : texts,
    notice: handled[m.modPath] ? m.modPath.split('/').pop() : `${m.modPath} ${m.version}`,
    partial: Boolean(handled[m.modPath]),
  });
}
for (const name of rendererPackages) {
  const p = npmPackage(name);
  targets.push({ folder: `npm/${p.name}`, texts: licenseTexts(p.dir), notice: `${p.name} ${p.version}` });
}

const problems = [];
const notice = readFileSync(path.join(root, 'NOTICE'), 'utf8');
for (const t of targets) {
  const dir = path.join(outRoot, t.folder);
  if (write && !t.partial) {
    rmSync(dir, { recursive: true, force: true });
    mkdirSync(dir, { recursive: true });
    for (const { name, text } of t.texts) {
      mkdirSync(path.dirname(path.join(dir, name)), { recursive: true });
      writeFileSync(path.join(dir, name), text);
    }
  }
  for (const { name, text } of t.texts) {
    const file = path.join(dir, name);
    if (!existsSync(file)) {
      problems.push(`missing ${path.relative(root, file)}`);
    } else if (readFileSync(file, 'utf8') !== text) {
      problems.push(`outdated ${path.relative(root, file)}`);
    }
  }
  if (!notice.includes(t.notice)) {
    problems.push(`NOTICE does not name "${t.notice}"`);
  }
}

// Folders left behind by modules that are no longer linked.
const expected = new Set(targets.map((t) => t.folder));
for (const kind of ['go', 'npm']) {
  const base = path.join(outRoot, kind);
  if (!existsSync(base)) continue;
  const walk = (dir) => {
    const entries = readdirSync(dir, { withFileTypes: true });
    if (entries.some((e) => e.isFile())) {
      const folder = path.relative(outRoot, dir).split(path.sep).join('/');
      if (!expected.has(folder)) {
        if (write) rmSync(dir, { recursive: true, force: true });
        else problems.push(`stale ${path.relative(root, dir)}`);
      }
      return;
    }
    for (const e of entries) walk(path.join(dir, e.name));
  };
  walk(base);
}

if (problems.length > 0) {
  console.error(problems.join('\n'));
  console.error(
    write
      ? '\nTexts written. Update NOTICE by hand for the lines above.'
      : '\nRun `node scripts/third-party-licenses.mjs --write` and update NOTICE.',
  );
  process.exit(1);
}
console.log(`third-party licenses ok (${targets.length} entries)`);
