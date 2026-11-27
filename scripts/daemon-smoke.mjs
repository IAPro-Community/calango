// Smoke do daemon calango no sistema nativo (issue #120). Sem shell específico de plataforma.
//
// Uso: node scripts/daemon-smoke.mjs [--binary <caminho>] [--summary <arquivo.md>]
// Sem --binary, compila ./cmd/calango num diretório temporário com o Go do PATH.
//
// Verifica, com data dir temporário (com espaço e acento no caminho) e CALANGO_VAULT_KEY:
//   1. print-token devolve um token não vazio e cria o banco no data dir;
//   2. no Unix, o banco nasce com permissão 0600 e o diretório do banco com 0700;
//   3. o daemon sobe e responde /healthz;
//   4. o token de print-token autentica na API e uma chamada sem token recebe 401;
//   5. o token não aparece na saída do daemon;
//   6. o encerramento (SIGTERM no Unix; TerminateProcess no Windows) termina o processo
//      dentro do prazo e libera a porta.
import { execFile, spawn } from 'node:child_process';
import { randomBytes } from 'node:crypto';
import { appendFile, mkdtemp, rm, stat } from 'node:fs/promises';
import net from 'node:net';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseArgs, promisify } from 'node:util';

const execFileAsync = promisify(execFile);
const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const isWindows = process.platform === 'win32';
const HEALTH_TIMEOUT_MS = 30_000;
const EXIT_TIMEOUT_MS = 10_000;

const { values } = parseArgs({
  options: {
    binary: { type: 'string' },
    summary: { type: 'string' },
  },
});

const results = [];

function record(name, ok, detail = '') {
  results.push({ name, ok, detail });
  console.log(`${ok ? 'ok  ' : 'FALHA'} ${name}${detail ? ` — ${detail}` : ''}`);
}

function check(name, condition, detail = '') {
  record(name, Boolean(condition), detail);
  if (!condition) {
    throw new Error(`${name}${detail ? `: ${detail}` : ''}`);
  }
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function freePort() {
  const server = net.createServer();
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const { port } = server.address();
  await new Promise((resolve) => server.close(resolve));
  return port;
}

async function portAcceptsConnections(port) {
  return new Promise((resolve) => {
    const socket = net.connect({ host: '127.0.0.1', port });
    socket.once('connect', () => {
      socket.destroy();
      resolve(true);
    });
    socket.once('error', () => resolve(false));
  });
}

async function buildBinary(dir) {
  const output = path.join(dir, isWindows ? 'calango.exe' : 'calango');
  await execFileAsync('go', ['build', '-o', output, './cmd/calango'], { cwd: repoRoot, maxBuffer: 16 * 1024 * 1024 });
  return output;
}

async function waitHealthy(baseUrl, child) {
  const deadline = Date.now() + HEALTH_TIMEOUT_MS;
  while (Date.now() < deadline) {
    if (child.exitCode !== null || child.signalCode !== null) {
      return false;
    }
    try {
      const res = await fetch(`${baseUrl}/healthz`, { signal: AbortSignal.timeout(1_000) });
      if (res.ok) {
        return true;
      }
    } catch {
      // Ainda não está escutando.
    }
    await delay(200);
  }
  return false;
}

function waitExit(child, timeoutMs) {
  if (child.exitCode !== null || child.signalCode !== null) {
    return Promise.resolve({ code: child.exitCode, signal: child.signalCode });
  }
  return new Promise((resolve) => {
    const timer = setTimeout(() => resolve(null), timeoutMs);
    child.once('exit', (code, signal) => {
      clearTimeout(timer);
      resolve({ code, signal });
    });
  });
}

async function writeSummary() {
  if (!values.summary) {
    return;
  }
  const lines = [
    `### Smoke do daemon — ${process.platform}/${process.arch}`,
    '',
    '| Verificação | Resultado | Detalhe |',
    '|---|---|---|',
    ...results.map((r) => `| ${r.name} | ${r.ok ? 'ok' : 'falha'} | ${r.detail.replaceAll('|', '\\|')} |`),
    '',
  ];
  await appendFile(values.summary, `${lines.join('\n')}\n`);
}

async function main() {
  const work = await mkdtemp(path.join(os.tmpdir(), 'calango-smoke-'));
  // Espaço e acento no caminho exercitam o tratamento de caminhos em cada sistema.
  const dataDir = path.join(work, 'dados do calango çá');
  let child = null;
  try {
    const binary = values.binary ? path.resolve(values.binary) : await buildBinary(work);
    const port = await freePort();
    const addr = `127.0.0.1:${port}`;
    const baseUrl = `http://${addr}`;
    const env = {
      ...process.env,
      CALANGO_ADDR: addr,
      CALANGO_DATA_DIR: dataDir,
      // Chave descartável gerada a cada execução: o smoke nunca toca o keyring do sistema.
      CALANGO_VAULT_KEY: randomBytes(32).toString('base64'),
    };

    const { stdout: tokenOut } = await execFileAsync(binary, ['print-token'], { env, timeout: 30_000 });
    const token = tokenOut.trim();
    check('print-token devolve token', token.length >= 16, `${token.length} caracteres`);

    const dbPath = path.join(dataDir, 'calango.db');
    const dbStat = await stat(dbPath);
    check('banco criado no data dir', dbStat.isFile(), path.basename(dataDir));
    if (isWindows) {
      record('permissões Unix do banco', true, 'não se aplica no Windows (ACL herdada do perfil)');
    } else {
      const dbMode = dbStat.mode & 0o777;
      const dirMode = (await stat(dataDir)).mode & 0o777;
      check('permissão do banco 0600', dbMode === 0o600, dbMode.toString(8));
      check('permissão do data dir 0700', dirMode === 0o700, dirMode.toString(8));
    }

    let output = '';
    child = spawn(binary, [], { env, stdio: ['ignore', 'pipe', 'pipe'] });
    child.stdout.on('data', (chunk) => {
      output += chunk;
    });
    child.stderr.on('data', (chunk) => {
      output += chunk;
    });
    const startedAt = Date.now();
    const healthy = await waitHealthy(baseUrl, child);
    check('daemon responde /healthz', healthy, healthy ? `${Date.now() - startedAt} ms` : output.slice(-2_000));

    const authed = await fetch(`${baseUrl}/v1/agents`, { headers: { Authorization: `Bearer ${token}` } });
    check('token autentica na API', authed.status === 200, `GET /v1/agents → ${authed.status}`);
    const anon = await fetch(`${baseUrl}/v1/agents`);
    check('chamada sem token é recusada', anon.status === 401, `GET /v1/agents → ${anon.status}`);

    const stopAt = Date.now();
    // No Windows, child.kill() usa TerminateProcess: não há SIGTERM para o daemon tratar.
    child.kill('SIGTERM');
    const exit = await waitExit(child, EXIT_TIMEOUT_MS);
    if (exit === null) {
      child.kill('SIGKILL');
      await waitExit(child, EXIT_TIMEOUT_MS);
    }
    check(
      'daemon encerra dentro do prazo',
      exit !== null,
      exit === null ? 'precisou de SIGKILL' : `code=${exit.code} signal=${exit.signal} em ${Date.now() - stopAt} ms`,
    );
    if (!isWindows) {
      // Com SIGTERM, o daemon cancela o contexto e faz shutdown limpo: sai com código 0.
      check('encerramento limpo com SIGTERM', exit.code === 0, `code=${exit.code}`);
    }
    child = null;
    check('porta liberada após encerrar', !(await portAcceptsConnections(port)), addr);
    check('token ausente da saída do daemon', !output.includes(token));
  } finally {
    if (child !== null) {
      child.kill('SIGKILL');
      await waitExit(child, EXIT_TIMEOUT_MS);
    }
    // No Windows, o SQLite pode segurar o arquivo por alguns instantes após o processo sair.
    await rm(work, { recursive: true, force: true, maxRetries: 5, retryDelay: 500 });
  }
}

try {
  await main();
} catch (error) {
  if (!results.some((r) => !r.ok)) {
    record('execução do smoke', false, error instanceof Error ? error.message : String(error));
  }
  process.exitCode = 1;
} finally {
  await writeSummary();
}
