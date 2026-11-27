// Playwright fixture that starts the real desktop app and the real daemon on a
// throwaway profile: temporary HOME, XDG config and data dirs, a random vault key, a
// free loopback port and the fake Codex CLI first on PATH. Nothing touches the
// person's profile, keyring or a real provider.
import { execFileSync } from 'node:child_process';
import { randomBytes } from 'node:crypto';
import { mkdtempSync, rmSync } from 'node:fs';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { test as base, _electron as electron, type ElectronApplication, type Page } from '@playwright/test';
import catalog from '../../../packages/i18n/messages.json' with { type: 'json' };

type MessageKey = keyof (typeof catalog)['en'];

const desktopRoot = path.resolve(__dirname, '..');
const repoRoot = path.resolve(desktopRoot, '..', '..');
export const fixturesDir = path.join(__dirname, 'fixtures');
export const FAKE_REPLY = 'Resposta determinística do provider fake.';

/** t returns the English interface text; the app runs with the en-US locale. */
export function t(key: MessageKey): string {
  return catalog.en[key];
}

export interface Desktop {
  app: ElectronApplication;
  page: Page;
  /** Values that must never reach the renderer or the logs. */
  secrets: { vaultKey: string };
  /** stdout and stderr of the Electron process, the daemon's included. */
  logs: () => string;
  /** api calls the daemon with the token the renderer received over IPC. */
  api: <T>(method: string, route: string, body?: unknown) => Promise<T>;
  /** daemonPid returns the pid of the daemon child, or undefined while there is none. */
  daemonPid: () => number | undefined;
}

async function freePort(): Promise<number> {
  const { promise, resolve, reject } = Promise.withResolvers<number>();
  const server = createServer();
  server.once('error', reject);
  server.listen(0, '127.0.0.1', () => {
    const address = server.address();
    server.close(() => resolve(typeof address === 'object' && address !== null ? address.port : 0));
  });
  return promise;
}

export const test = base.extend<{ desktop: Desktop }>({
  // biome-ignore lint/correctness/noEmptyPattern: Playwright reads fixture dependencies from this parameter.
  desktop: async ({}, use, testInfo) => {
    const profile = mkdtempSync(path.join(tmpdir(), 'calango-e2e-'));
    const port = await freePort();
    const vaultKey = randomBytes(32).toString('base64');
    let output = '';

    const app = await electron.launch({
      args: [desktopRoot, '--lang=en-US'],
      cwd: desktopRoot,
      env: {
        PATH: `${path.join(fixturesDir, 'bin')}${path.delimiter}${process.env.PATH ?? ''}`,
        HOME: profile,
        XDG_CONFIG_HOME: path.join(profile, 'config'),
        XDG_DATA_HOME: path.join(profile, 'data'),
        XDG_CACHE_HOME: path.join(profile, 'cache'),
        CODEX_HOME: path.join(profile, 'codex'),
        CALANGO_ADDR: `127.0.0.1:${port}`,
        CALANGO_DATA_DIR: path.join(profile, 'calango'),
        CALANGO_VAULT_KEY: vaultKey,
        CALANGO_BIN: path.join(repoRoot, 'bin', 'calango'),
        LANG: 'en_US.UTF-8',
        ...(process.env.DISPLAY ? { DISPLAY: process.env.DISPLAY } : {}),
        ...(process.env.XAUTHORITY ? { XAUTHORITY: process.env.XAUTHORITY } : {}),
        ...(process.env.WAYLAND_DISPLAY ? { WAYLAND_DISPLAY: process.env.WAYLAND_DISPLAY } : {}),
        ...(process.env.XDG_RUNTIME_DIR ? { XDG_RUNTIME_DIR: process.env.XDG_RUNTIME_DIR } : {}),
      },
    });
    const child = app.process();
    child.stdout?.on('data', (chunk: Buffer) => {
      output += chunk.toString();
    });
    child.stderr?.on('data', (chunk: Buffer) => {
      output += chunk.toString();
    });

    // Traces and screenshots are kept only for failures; every value in them is fake.
    await app.context().tracing.start({ screenshots: true, snapshots: true });
    const page = await app.firstWindow();

    const desktop: Desktop = {
      app,
      page,
      secrets: { vaultKey },
      logs: () => output,
      api: async (method, route, body) => {
        const conn = await page.evaluate(() => window.calango.getConnection());
        const res = await fetch(`${conn.baseUrl}${route}`, {
          method,
          headers: { Authorization: `Bearer ${conn.token}`, 'Content-Type': 'application/json' },
          body: body === undefined ? undefined : JSON.stringify(body),
        });
        if (!res.ok) {
          throw new Error(`${method} ${route}: ${res.status} ${await res.text()}`);
        }
        return (res.status === 204 ? undefined : await res.json()) as never;
      },
      daemonPid: () => {
        try {
          const out = execFileSync('pgrep', ['-P', String(child.pid), '-x', 'calango'], { encoding: 'utf8' });
          const pid = Number(out.trim().split('\n')[0]);
          return Number.isInteger(pid) && pid > 0 ? pid : undefined;
        } catch {
          return undefined;
        }
      },
    };

    try {
      await use(desktop);
    } finally {
      const failed = testInfo.status !== testInfo.expectedStatus;
      if (failed) {
        await page
          .screenshot({ path: testInfo.outputPath('failure.png') })
          .then(() =>
            testInfo.attachments.push({
              name: 'screenshot',
              path: testInfo.outputPath('failure.png'),
              contentType: 'image/png',
            }),
          )
          .catch(() => {});
        await app.context().tracing.stop({ path: testInfo.outputPath('trace.zip') });
        await testInfo.attach('electron.log', { body: output, contentType: 'text/plain' });
      } else {
        await app.context().tracing.stop();
      }
      await app.close();
      rmSync(profile, { recursive: true, force: true });
    }
  },
});

export { expect } from '@playwright/test';

/**
 * onboard creates the first provider account and agent through the interface, as a
 * person would on a fresh install.
 */
export async function onboard(page: Page, agentName = 'Agente E2E'): Promise<void> {
  await page.getByRole('button', { name: t('navAgents'), exact: true }).click();
  const accountForm = page.locator('form').filter({ has: page.getByLabel(t('accountKind')) });
  await accountForm.getByLabel(t('accountKind')).selectOption('codex');
  await accountForm.getByRole('button', { name: t('accountCreate') }).click();
  await page.locator('.list li').filter({ hasText: 'codex/default' }).waitFor();

  const agentForm = page.locator('form').filter({ has: page.getByLabel(t('agentName')) });
  await agentForm.getByLabel(t('agentName')).fill(agentName);
  await agentForm.getByLabel(t('agentModel')).fill('gpt-fake');
  await agentForm.getByRole('button', { name: t('agentCreate') }).click();
  await page.locator('.list li').filter({ hasText: agentName }).waitFor();
}

/** startConversation opens a new conversation with the first agent. */
export async function startConversation(page: Page, title: string): Promise<void> {
  await page.getByRole('button', { name: t('navConversations'), exact: true }).click();
  await page.getByLabel(t('conversationTitleLabel')).fill(title);
  await page.getByRole('button', { name: t('actionNewConversation') }).click();
  await page.getByRole('button', { name: title }).waitFor();
}

/** send types a message in the open conversation and sends it. */
export async function send(page: Page, body: string): Promise<void> {
  await page.locator('.composer textarea').fill(body);
  await page
    .locator('.composer')
    .getByRole('button', { name: t('actionSend') })
    .click();
}
