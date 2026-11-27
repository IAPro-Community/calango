// Compila o daemon calango para o sistema e a arquitetura pedidos, sem cgo e com caminhos
// removidos do binário, para que o mesmo commit gere o mesmo executável.
// Uso: node scripts/build-daemon.mjs --goos windows --goarch amd64 --out bin/calango.exe
import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseArgs } from 'node:util';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const { values } = parseArgs({
  options: {
    goos: { type: 'string' },
    goarch: { type: 'string' },
    out: { type: 'string' },
  },
});
if (!values.goos || !values.goarch || !values.out) {
  console.error('Uso: node scripts/build-daemon.mjs --goos <os> --goarch <arch> --out <arquivo>');
  process.exit(2);
}

execFileSync(
  'go',
  ['build', '-trimpath', '-ldflags=-s -w -buildid=', '-o', path.resolve(repoRoot, values.out), './cmd/calango'],
  {
    cwd: repoRoot,
    stdio: 'inherit',
    env: { ...process.env, CGO_ENABLED: '0', GOOS: values.goos, GOARCH: values.goarch },
  },
);
