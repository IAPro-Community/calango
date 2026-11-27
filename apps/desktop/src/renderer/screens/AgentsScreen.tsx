import { useCallback, useEffect, useState, type FormEvent } from 'react';
import type { Agent, ProviderAccount } from '@calango/sdk-ts';
import { usePoll } from '../usePoll';
import type { ScreenProps } from './props';
import { Button, Card, Field, Input, PageHeader, Select, Textarea } from '@';

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
    <section className="flex flex-col gap-md">
      <PageHeader
        title={t('navAgents')}
        actions={
          <>
            <Button
              variant="ghost"
              onClick={() => {
                setActionError(null);
                void client
                  .pauseAll()
                  .then(refresh)
                  .catch((err) => setActionError(err instanceof Error ? err.message : String(err)));
              }}
            >
              {t('agentPauseAll')}
            </Button>
            <Button
              variant="ghost"
              onClick={() => {
                setActionError(null);
                void client
                  .resumeAll()
                  .then(refresh)
                  .catch((err) => setActionError(err instanceof Error ? err.message : String(err)));
              }}
            >
              {t('agentResumeAll')}
            </Button>
          </>
        }
      />
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {actionError && (
        <p className="error" role="alert">
          {actionError}
        </p>
      )}

      <div className="grid grid-cols-[repeat(auto-fit,minmax(320px,1fr))] items-start gap-md">
        <Card title={t('agentCreate')}>
          <AgentForm accounts={accounts} onCreated={refresh} client={client} t={t} />
        </Card>
        <Card title={t('accountCreate')}>
          <AccountForm onCreated={refresh} client={client} t={t} />
          <ul className="list">
            {accounts.map((account) => (
              <li key={account.id}>
                <span>{account.key}</span>
                <span className="muted">{account.setup}</span>
              </li>
            ))}
          </ul>
        </Card>
      </div>

      <Card>
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
                    <Button
                      variant="ghost"
                      onClick={() => void toggleAgentMaestro(agent)}
                      disabled={maestroBusyAgent !== null}
                    >
                      {maestroBusyAgent === agent.id
                        ? t('commonLoading')
                        : t(agent.maestro_enabled ? 'agentMaestroUnlink' : 'agentMaestroLink')}
                    </Button>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </Card>
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
    <form className="flex flex-col gap-sm" onSubmit={(event) => void submit(event)}>
      <Field label={t('accountKind')}>
        <Select value={kind} onChange={(event) => setKind(event.target.value as 'codex' | 'anthropic')}>
          <option value="codex">Codex</option>
          <option value="anthropic">Anthropic</option>
        </Select>
      </Field>
      <Field label={t('accountLabel')}>
        <Input
          value={label}
          onChange={(event) => setLabel(event.target.value)}
          required
          pattern="[a-z0-9][a-z0-9-]{0,31}"
        />
      </Field>
      {kind === 'anthropic' ? (
        <Field label={t('accountAnthropicKey')}>
          <Input
            type="password"
            value={secret}
            onChange={(event) => setSecret(event.target.value)}
            required
            autoComplete="off"
          />
        </Field>
      ) : (
        <p className="muted">{t('accountCodexNote')}</p>
      )}
      <p className="muted">{t('accountSecretNote')}</p>
      {formError && (
        <p className="error" role="alert">
          {formError}
        </p>
      )}
      <Button type="submit" disabled={busy}>
        {t('accountCreate')}
      </Button>
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
    <form className="flex flex-col gap-sm" onSubmit={(event) => void submit(event)}>
      <Field label={t('agentName')}>
        <Input value={name} onChange={(event) => setName(event.target.value)} required />
      </Field>
      <Field label={t('agentRole')}>
        <Input value={role} onChange={(event) => setRole(event.target.value)} />
      </Field>
      <Field label={t('agentInstructions')}>
        <Textarea value={instructions} onChange={(event) => setInstructions(event.target.value)} rows={3} />
      </Field>
      <Field label={t('agentProviderAccount')}>
        <Select value={accountId} onChange={(event) => setAccountId(event.target.value)} required>
          <option value="" disabled>
            —
          </option>
          {accounts.map((account) => (
            <option key={account.id} value={account.id}>
              {account.key}
            </option>
          ))}
        </Select>
      </Field>
      <Field label={t('agentModel')}>
        <Input value={model} onChange={(event) => setModel(event.target.value)} required />
      </Field>
      <Field label={t('agentTokenLimit')}>
        <Input
          type="number"
          min={1}
          value={tokenLimit}
          onChange={(event) => setTokenLimit(Number(event.target.value))}
          required
        />
      </Field>
      <Field label={t('agentRunOn')}>
        <Select value={runOn} onChange={(event) => setRunOn(event.target.value as 'auto' | 'pc' | 'vps')}>
          <option value="auto">{t('runOnAuto')}</option>
          <option value="pc">{t('runOnPc')}</option>
          <option value="vps">{t('runOnVps')}</option>
        </Select>
      </Field>
      <Field label={t('agentApprovalLevel')}>
        <Select
          value={approvalLevel}
          onChange={(event) => setApprovalLevel(event.target.value as 'always' | 'sensitive' | 'trusted')}
        >
          <option value="always">{t('approvalAlways')}</option>
          <option value="sensitive">{t('approvalSensitive')}</option>
          <option value="trusted">{t('approvalTrusted')}</option>
        </Select>
      </Field>
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
      {formError && (
        <p className="error" role="alert">
          {formError}
        </p>
      )}
      <Button type="submit" disabled={busy}>
        {t('agentCreate')}
      </Button>
    </form>
  );
}
