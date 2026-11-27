// Desktop E2E: the packaged-shape app (dist/) with the real daemon and the fake
// Codex CLI. Each test gets a fresh profile, database and port.
import path from 'node:path';
import { expect, FAKE_REPLY, fixturesDir, onboard, send, startConversation, t, test } from './app';

test('onboarding cria conta, agente e mantém a fronteira IPC fechada', async ({ desktop }) => {
  const { page } = desktop;
  await expect(page.getByRole('heading', { name: t('navAgents') })).toBeVisible();
  await expect(page.getByText(t('agentNeedsAccount'))).toBeVisible();

  await onboard(page);
  await expect(page.locator('.list li').filter({ hasText: 'Agente E2E' })).toContainText(t('agentStatusActive'));

  // The renderer sees only the bridge: no Node, no ipcRenderer, no extra methods.
  const surface = await page.evaluate(() => ({
    bridge: Object.keys(window.calango).sort(),
    require: typeof (window as { require?: unknown }).require,
    process: typeof (window as { process?: unknown }).process,
    ipcRenderer: typeof (window as { ipcRenderer?: unknown }).ipcRenderer,
  }));
  expect(surface).toEqual({
    bridge: [
      'browser',
      'getConnection',
      'getLoginItemSettings',
      'getState',
      'onPtyClosed',
      'onPtyOutput',
      'onQuickConversation',
      'onShortcutUnavailable',
      'onState',
      'ptyAttach',
      'ptyDetach',
      'ptyInput',
      'ptyResize',
      'setLaunchAtLogin',
    ],
    require: 'undefined',
    process: 'undefined',
    ipcRenderer: 'undefined',
  });
  // The main process validates IPC arguments.
  await expect(page.evaluate(() => window.calango.setLaunchAtLogin('yes' as unknown as boolean))).rejects.toThrow(
    /must be a boolean/,
  );
  // New windows and navigation away from the app origin are refused.
  expect(await page.evaluate(() => window.open('https://example.com') === null)).toBe(true);
  await page.evaluate(() => {
    window.location.href = 'https://example.com/';
  });
  await page.waitForTimeout(500);
  expect(page.url()).toBe('app://calango/');
  expect(desktop.app.windows()).toHaveLength(1);
});

test('chat responde pelo provider fake e o uso aparece', async ({ desktop }) => {
  const { page } = desktop;
  await onboard(page);
  await startConversation(page, 'Conversa E2E');
  await send(page, 'Olá, agente');

  await expect(page.locator('.message').filter({ hasText: 'Olá, agente' })).toBeVisible();
  await expect(page.locator('.message').filter({ hasText: FAKE_REPLY })).toBeVisible();
  await expect(page.locator('.run')).toContainText(t('runDone'));

  await page.getByRole('button', { name: t('navUsage'), exact: true }).click();
  const row = page.locator('table tbody tr').first();
  await expect(row).toBeVisible();
  const tokensIn = Number(await row.locator('td').nth(1).innerText());
  const tokensOut = Number(await row.locator('td').nth(2).innerText());
  expect(tokensIn).toBeGreaterThan(0);
  expect(tokensOut).toBeGreaterThan(0);
});

test('cancelar interrompe uma execução em andamento', async ({ desktop }) => {
  const { page } = desktop;
  await onboard(page);
  await startConversation(page, 'Cancelamento');
  await send(page, 'Demore bastante [e2e:slow]');

  const run = page.locator('.run');
  await expect(run).toContainText(t('runRunning'));
  await run.getByRole('button', { name: t('actionStop') }).click();
  await expect(run).toContainText(t('runStopped'));
  await expect(run.getByRole('button', { name: t('actionStop') })).toHaveCount(0);
  await expect(page.locator('.message').filter({ hasText: FAKE_REPLY })).toHaveCount(0);
});

test('escrita por conector espera aprovação e segue depois de aprovada', async ({ desktop }) => {
  const { page, api } = desktop;
  await onboard(page);

  // Connector setup has no screen yet, so it goes through the same local API the app uses.
  const [agent] = await api<{ id: string }[]>('GET', '/v1/agents');
  const connector = await api<{ id: string }>('POST', '/v1/connectors', {
    name: 'notas',
    transport: 'stdio',
    command: process.execPath,
    args: [path.join(fixturesDir, 'fake-mcp.mjs')],
  });
  await api('PUT', `/v1/agents/${agent.id}/tools/${connector.id}`, { access: 'write' });

  await startConversation(page, 'Aprovação');
  await send(page, 'Salve uma nota [e2e:tool]');

  await expect(page.locator('.run')).toContainText(t('runWaitingApproval'));
  const approval = page.locator('li.approval');
  await expect(approval).toContainText('notas/create_note');
  await approval.getByRole('button', { name: t('actionApprove'), exact: true }).click();

  await expect(page.locator('.message').filter({ hasText: 'nota salva: nota do teste E2E' })).toBeVisible();
  await expect(page.locator('.run')).toContainText(t('runDone'));

  await page.getByRole('button', { name: t('navApprovals'), exact: true }).click();
  await expect(page.getByText(t('approvalEmpty'))).toBeVisible();
});

test('rotina é criada, pausada e retomada', async ({ desktop }) => {
  const { page } = desktop;
  await onboard(page);
  await page.getByRole('button', { name: t('navRoutines'), exact: true }).click();
  await expect(page.getByText(t('routineEmpty'))).toBeVisible();

  const form = page.locator('form');
  await form.getByLabel(t('routineName')).fill('Resumo diário');
  await form.getByLabel(t('routinePrompt')).fill('Resuma o dia');
  await form.getByLabel(t('routineTrigger')).selectOption('interval');
  await form.getByLabel(t('routineSchedule'), { exact: true }).fill('1h');
  await form.getByRole('button', { name: t('routineCreate') }).click();

  const item = page.locator('.list li').filter({ hasText: 'Resumo diário' });
  await expect(item).toContainText('1h');
  await item.getByRole('button', { name: t('routinePause') }).click();
  await expect(item.getByRole('button', { name: t('routineResume') })).toBeVisible();
  await item.getByRole('button', { name: t('routineResume') }).click();
  await expect(item.getByRole('button', { name: t('routinePause') })).toBeVisible();
});

test('app reinicia o daemon depois de uma queda sem perder dados', async ({ desktop }) => {
  const { page } = desktop;
  await onboard(page);
  const first = desktop.daemonPid();
  expect(first).toBeDefined();

  process.kill(first as number, 'SIGKILL');
  await expect(page.getByRole('heading', { name: t('daemonRetrying') })).toBeVisible();

  await expect(page.getByRole('heading', { name: t('navAgents') })).toBeVisible({ timeout: 20_000 });
  await expect(page.locator('.list li').filter({ hasText: 'Agente E2E' })).toBeVisible();
  const second = desktop.daemonPid();
  expect(second).toBeDefined();
  expect(second).not.toBe(first);

  // The fresh token from the restarted daemon works.
  await startConversation(page, 'Depois da queda');
  await send(page, 'Ainda está aí?');
  await expect(page.locator('.message').filter({ hasText: FAKE_REPLY })).toBeVisible();
});

test('segredos não chegam ao renderer nem aos logs', async ({ desktop }) => {
  const { page, api } = desktop;
  const providerKey = `sk-ant-e2e-${Date.now().toString(36)}-fake`;
  await page.getByRole('button', { name: t('navAgents'), exact: true }).click();
  const accountForm = page.locator('form').filter({ has: page.getByLabel(t('accountKind')) });
  await accountForm.getByLabel(t('accountKind')).selectOption('anthropic');
  await accountForm.getByLabel(t('accountAnthropicKey')).fill(providerKey);
  await accountForm.getByRole('button', { name: t('accountCreate') }).click();
  await expect(page.locator('.list li').filter({ hasText: 'anthropic/default' })).toBeVisible();
  await expect(accountForm.getByLabel(t('accountAnthropicKey'))).toHaveValue('');

  // The API lists the vault item by name only.
  const items = await api<unknown[]>('GET', '/v1/vault');
  expect(JSON.stringify(items)).not.toContain(providerKey);

  const rendererState = await page.evaluate(() =>
    JSON.stringify({
      html: document.documentElement.outerHTML,
      local: { ...localStorage },
      session: { ...sessionStorage },
    }),
  );
  for (const secret of [providerKey, desktop.secrets.vaultKey]) {
    expect(rendererState).not.toContain(secret);
    expect(desktop.logs()).not.toContain(secret);
  }
  const conn = await page.evaluate(() => window.calango.getConnection());
  expect(desktop.logs()).not.toContain(conn.token);
});
