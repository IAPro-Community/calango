import { useCallback, useEffect, useState, type FormEvent } from 'react';
import type { Agent, ProviderAccount } from '@calango/sdk-ts';
import { usePoll } from '../usePoll';
import type { ScreenProps } from './props';

const DEFAULT_LABEL = 'default';

// The vault item for an Anthropic account. Mirrors providers.keyItem in the daemon: the
// API returns only the account key (kind/label), not the item name.
function anthropicKeyItem(label: string): string {
  return label === DEFAULT_LABEL ? 'anthropic-api-key' : `anthropic-${label}-api-key`;
}

export function AgentsScreen({ client, t }: ScreenProps) {
  const load = useCallback(async () => {
    const [agents, accounts] = await Promise.all([client.listAgents(), client.listProviderAccounts()]);
    return { agents, accounts };
  }, [client]);
  const { data, error, refresh } = usePoll(load, 5_000);
  const agents = data?.agents ?? [];
  const accounts = data?.accounts ?? [];

  const [actionError, setActionError] = useState<string | null>(null);
  const [maestroBusyAgent, setMaestroBusyAgent] = useState<string | null>(null);

  async function toggleAgentMaestro(agent: Agent) {
    setActionError(null);
    setMaestroBusyAgent(agent.id);
    try {
      await client.updateAgent(agent.id, {
        name: agent.name,
        role: agent.role || undefined,
        instructions: agent.instructions || undefined,
        providers: agent.providers.map(({ account_id, model }) => ({ account_id, model })),
        reasoning_effort: agent.reasoning_effort || undefined,
        run_on: agent.run_on,
        token_limit_month: agent.token_limit_month,
        approval_level: agent.approval_level,
        maestro_enabled: !agent.maestro_enabled,
      });
      refresh();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : String(err));
    } finally {
      setMaestroBusyAgent(null);
    }
  }

  return (
    <section className="screen">
      <header className="screen-header">
        <h1>{t('navAgents')}</h1>
        <div className="actions">
          <button
            type="button"
            onClick={() => {
              setActionError(null);
              void client
                .pauseAll()
                .then(refresh)
                .catch((err) => setActionError(err instanceof Error ? err.message : String(err)));
            }}
          >
            {t('agentPauseAll')}
          </button>
          <button
            type="button"
            onClick={() => {
              setActionError(null);
              void client
                .resumeAll()
                .then(refresh)
                .catch((err) => setActionError(err instanceof Error ? err.message : String(err)));
            }}
          >
            {t('agentResumeAll')}
          </button>
        </div>
      </header>
      {error && <p className="error">{error}</p>}
      {actionError && <p className="error">{actionError}</p>}

      <div className="grid">
        <div className="card">
          <h2>{t('agentCreate')}</h2>
          <AgentForm accounts={accounts} onCreated={refresh} client={client} t={t} />
        </div>
        <div className="card">
          <h2>{t('accountCreate')}</h2>
          <AccountForm onCreated={refresh} client={client} t={t} />
          <ul className="list">
            {accounts.map((account) => (
              <li key={account.id}>
                <span>{account.key}</span>
                <span className="muted">{account.setup}</span>
              </li>
            ))}
          </ul>
        </div>
      </div>

      <div className="card">
        {agents.length === 0 ? (
          <p className="muted">{t('agentEmpty')}</p>
        ) : (
          <ul className="list">
            {agents.map((agent: Agent) => {
              const hasCodex = agent.providers.some((provider) => provider.account?.startsWith('codex/'));
              return (
                <li key={agent.id}>
                  <span>{agent.name}</span>
                  <span className="muted">
                    {agent.role ? `${agent.role} · ` : ''}
                    {t(agent.paused ? 'agentStatusPaused' : 'agentStatusActive')}
                  </span>
                  {hasCodex && (
                    <button
                      type="button"
                      onClick={() => void toggleAgentMaestro(agent)}
                      disabled={maestroBusyAgent !== null}
                    >
                      {maestroBusyAgent === agent.id
                        ? t('commonLoading')
                        : t(agent.maestro_enabled ? 'agentMaestroUnlink' : 'agentMaestroLink')}
                    </button>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
}

function AccountForm({ client, t, onCreated }: ScreenProps & { onCreated: () => void }) {
  const [kind, setKind] = useState<'codex' | 'anthropic'>('codex');
  const [label, setLabel] = useState(DEFAULT_LABEL);
  const [secret, setSecret] = useState('');
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setFormError(null);
    try {
      await client.createProviderAccount({ kind, label });
      if (kind === 'anthropic') {
        await client.setVaultSecret(anthropicKeyItem(label), secret);
        setSecret('');
      }
      onCreated();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="form" onSubmit={(event) => void submit(event)}>
      <label>
        {t('accountKind')}
        <select value={kind} onChange={(event) => setKind(event.target.value as 'codex' | 'anthropic')}>
          <option value="codex">Codex</option>
          <option value="anthropic">Anthropic</option>
        </select>
      </label>
      <label>
        {t('accountLabel')}
        <input
          value={label}
          onChange={(event) => setLabel(event.target.value)}
          required
          pattern="[a-z0-9][a-z0-9-]{0,31}"
        />
      </label>
      {kind === 'anthropic' ? (
        <label>
          {t('accountAnthropicKey')}
          <input
            type="password"
            value={secret}
            onChange={(event) => setSecret(event.target.value)}
            required
            autoComplete="off"
          />
        </label>
      ) : (
        <p className="muted">{t('accountCodexNote')}</p>
      )}
      <p className="muted">{t('accountSecretNote')}</p>
      {formError && <p className="error">{formError}</p>}
      <button type="submit" disabled={busy}>
        {t('accountCreate')}
      </button>
    </form>
  );
}

function AgentForm({
  client,
  t,
  accounts,
  onCreated,
}: ScreenProps & { accounts: ProviderAccount[]; onCreated: () => void }) {
  const [name, setName] = useState('');
  const [role, setRole] = useState('');
  const [instructions, setInstructions] = useState('');
  const [accountId, setAccountId] = useState('');
  const [model, setModel] = useState('');
  const [tokenLimit, setTokenLimit] = useState(1_000_000);
  const [runOn, setRunOn] = useState<'auto' | 'pc' | 'vps'>('auto');
  const [approvalLevel, setApprovalLevel] = useState<'always' | 'sensitive' | 'trusted'>('sensitive');
  const [maestroEnabled, setMaestroEnabled] = useState(false);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    if (accountId === '' && accounts[0]) {
      setAccountId(accounts[0].id);
    }
  }, [accountId, accounts]);
  const selectedAccount = accounts.find((account) => account.id === accountId);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setFormError(null);
    try {
      await client.createAgent({
        name,
        role: role || undefined,
        instructions: instructions || undefined,
        providers: [{ account_id: accountId, model }],
        token_limit_month: tokenLimit,
        run_on: runOn,
        approval_level: approvalLevel,
        maestro_enabled: selectedAccount?.kind === 'codex' && maestroEnabled,
      });
      setName('');
      setRole('');
      setInstructions('');
      onCreated();
      setMaestroEnabled(false);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  if (accounts.length === 0) {
    return <p className="muted">{t('agentNeedsAccount')}</p>;
  }

  return (
    <form className="form" onSubmit={(event) => void submit(event)}>
      <label>
        {t('agentName')}
        <input value={name} onChange={(event) => setName(event.target.value)} required />
      </label>
      <label>
        {t('agentRole')}
        <input value={role} onChange={(event) => setRole(event.target.value)} />
      </label>
      <label>
        {t('agentInstructions')}
        <textarea value={instructions} onChange={(event) => setInstructions(event.target.value)} rows={3} />
      </label>
      <label>
        {t('agentProviderAccount')}
        <select value={accountId} onChange={(event) => setAccountId(event.target.value)} required>
          <option value="" disabled>
            —
          </option>
          {accounts.map((account) => (
            <option key={account.id} value={account.id}>
              {account.key}
            </option>
          ))}
        </select>
      </label>
      <label>
        {t('agentModel')}
        <input value={model} onChange={(event) => setModel(event.target.value)} required />
      </label>
      <label>
        {t('agentTokenLimit')}
        <input
          type="number"
          min={1}
          value={tokenLimit}
          onChange={(event) => setTokenLimit(Number(event.target.value))}
          required
        />
      </label>
      <label>
        {t('agentRunOn')}
        <select value={runOn} onChange={(event) => setRunOn(event.target.value as 'auto' | 'pc' | 'vps')}>
          <option value="auto">{t('runOnAuto')}</option>
          <option value="pc">{t('runOnPc')}</option>
          <option value="vps">{t('runOnVps')}</option>
        </select>
      </label>
      <label>
        {t('agentApprovalLevel')}
        <select
          value={approvalLevel}
          onChange={(event) => setApprovalLevel(event.target.value as 'always' | 'sensitive' | 'trusted')}
        >
          <option value="always">{t('approvalAlways')}</option>
          <option value="sensitive">{t('approvalSensitive')}</option>
          <option value="trusted">{t('approvalTrusted')}</option>
        </select>
      </label>
      {selectedAccount?.kind === 'codex' && (
        <>
          <label className="check-label">
            <input
              type="checkbox"
              checked={maestroEnabled}
              onChange={(event) => setMaestroEnabled(event.target.checked)}
            />
            {t('agentMaestroLink')}
          </label>
          <p className="muted">{t('agentMaestroDescription')}</p>
        </>
      )}
      {formError && <p className="error">{formError}</p>}
      <button type="submit" disabled={busy}>
        {t('agentCreate')}
      </button>
    </form>
  );
}
