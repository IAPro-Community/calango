// Resume a saída de `go test -json` em Markdown para o $GITHUB_STEP_SUMMARY (issue #120).
//
// Uso: node scripts/go-test-summary.mjs <go-test.json> [título]
// Escreve na saída padrão: contagem de testes, falhas e cada skip com o motivo registrado.
import { readFile } from 'node:fs/promises';

const [file, title = `Testes Go — ${process.platform}/${process.arch}`] = process.argv.slice(2);
if (!file) {
  console.error('uso: node scripts/go-test-summary.mjs <go-test.json> [título]');
  process.exit(2);
}

const outputs = new Map();
const counts = { pass: 0, fail: 0, skip: 0 };
const failed = [];
const skipped = [];
const failedPackages = [];
let unparsed = 0;

for (const line of (await readFile(file, 'utf8')).split(/\r?\n/)) {
  if (line.trim() === '') continue;
  let event;
  try {
    event = JSON.parse(line);
  } catch {
    unparsed++;
    continue;
  }
  const key = `${event.Package} ${event.Test ?? ''}`;
  if (event.Action === 'output') {
    outputs.set(key, `${outputs.get(key) ?? ''}${event.Output}`);
    continue;
  }
  if (!['pass', 'fail', 'skip'].includes(event.Action)) continue;
  if (!event.Test) {
    if (event.Action === 'fail') failedPackages.push(event.Package);
    continue;
  }
  counts[event.Action]++;
  if (event.Action === 'fail') failed.push(`${event.Package}.${event.Test}`);
  if (event.Action === 'skip') {
    const reason =
      (outputs.get(key) ?? '')
        .split('\n')
        .map((l) => l.trim())
        .filter((l) => l !== '' && !l.startsWith('=== ') && !l.startsWith('--- SKIP'))
        .pop() ?? 'sem motivo registrado';
    skipped.push({ name: `${event.Package}.${event.Test}`, reason });
  }
}

const escapeCell = (value) => value.replaceAll('|', '\\|');
const lines = [`### ${title}`, '', `Passaram: ${counts.pass} · Falharam: ${counts.fail} · Pulados: ${counts.skip}`, ''];
if (failedPackages.length > 0) {
  lines.push(`Pacotes com falha: ${failedPackages.map((p) => `\`${p}\``).join(', ')}`, '');
}
if (failed.length > 0) {
  lines.push('Testes com falha:', '', ...failed.map((name) => `- \`${name}\``), '');
}
if (skipped.length > 0) {
  lines.push(
    '| Teste pulado | Motivo |',
    '|---|---|',
    ...skipped.map((s) => `| \`${s.name}\` | ${escapeCell(s.reason)} |`),
    '',
  );
}
if (unparsed > 0) {
  lines.push(`Linhas fora do formato JSON (erro de compilação ou saída do runner): ${unparsed}.`, '');
}
console.log(lines.join('\n'));
