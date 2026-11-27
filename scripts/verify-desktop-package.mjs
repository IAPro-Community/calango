// Confere o conteúdo legal e o daemon de um pacote do desktop já montado (issue #40).
// Falha se LICENSE, NOTICE ou qualquer arquivo de third_party/ (incluindo iapro-sdk e agenticos)
// faltar no pacote ou divergir do repositório, ou se o daemon não estiver lá (e executável fora
// do Windows). Uso:
//   node scripts/verify-desktop-package.mjs --resources <pasta resources do app> [--daemon calango]
import { constants } from 'node:fs';
import { access, readdir, readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseArgs } from 'node:util';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

// Arquivos que precisam estar no pacote mesmo que third_party/ mude de forma (comentário da #40).
export const requiredLegalFiles = [
  'LICENSE',
  'NOTICE',
  'third_party/iapro-sdk/LICENSE',
  'third_party/iapro-sdk/NOTICE',
  'third_party/agenticos/LICENSE',
  'third_party/agenticos/NOTICE',
];

async function listFiles(dir, prefix) {
  const out = [];
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const rel = path.posix.join(prefix, entry.name);
    if (entry.isDirectory()) out.push(...(await listFiles(path.join(dir, entry.name), rel)));
    else out.push(rel);
  }
  return out;
}

// expectedLegalFiles lista o que o pacote precisa levar: LICENSE, NOTICE e todo third_party/.
export async function expectedLegalFiles(root = repoRoot) {
  const files = new Set(['LICENSE', 'NOTICE', ...(await listFiles(path.join(root, 'third_party'), 'third_party'))]);
  for (const f of requiredLegalFiles) files.add(f);
  return [...files].sort();
}

// verifyPackage devolve a lista de problemas encontrados; vazia significa pacote válido.
export async function verifyPackage({ resources, daemon = 'calango', root = repoRoot, platform = process.platform }) {
  const problems = [];
  for (const rel of await expectedLegalFiles(root)) {
    const source = path.join(root, rel);
    const packaged = path.join(resources, 'legal', rel);
    let want;
    try {
      want = await readFile(source);
    } catch {
      problems.push(`ausente no repositório: ${rel}`);
      continue;
    }
    let got;
    try {
      got = await readFile(packaged);
    } catch {
      problems.push(`ausente no pacote: legal/${rel}`);
      continue;
    }
    if (!want.equals(got)) problems.push(`diferente do repositório: legal/${rel}`);
  }

  const daemonPath = path.join(resources, 'bin', daemon);
  try {
    const info = await stat(daemonPath);
    if (!info.isFile() || info.size === 0) problems.push(`daemon inválido: bin/${daemon}`);
    else if (platform !== 'win32' && !daemon.endsWith('.exe')) {
      try {
        await access(daemonPath, constants.X_OK);
      } catch {
        problems.push(`daemon sem permissão de execução: bin/${daemon}`);
      }
    }
  } catch {
    problems.push(`ausente no pacote: bin/${daemon}`);
  }
  return problems;
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const { values } = parseArgs({ options: { resources: { type: 'string' }, daemon: { type: 'string' } } });
  if (!values.resources) {
    console.error('Uso: node scripts/verify-desktop-package.mjs --resources <pasta> [--daemon calango]');
    process.exit(2);
  }
  const resources = path.resolve(values.resources);
  const problems = await verifyPackage({ resources, daemon: values.daemon });
  if (problems.length > 0) {
    console.error(`Pacote do desktop inválido (${resources}):`);
    for (const p of problems) console.error(`- ${p}`);
    process.exit(1);
  }
  console.log(`Pacote do desktop ok: LICENSE, NOTICE, third_party/ e bin/${values.daemon ?? 'calango'} (${resources})`);
}
