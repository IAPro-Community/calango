import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import type { Agent, Approval, Conversation, Message, Run } from '@calango/sdk-ts';
import { usePoll } from '../usePoll';
import type { MessageKey } from '../i18n';
import type { ScreenProps } from './props';

type RunStatus = NonNullable<Run['status']>;

const RUN_STATUS_KEY: Record<RunStatus, MessageKey> = {
  queued: 'runQueued',
  running: 'runRunning',
  waiting_approval: 'runWaitingApproval',
  done: 'runDone',
  failed: 'runFailed',
  stopped: 'runStopped',
};

const TERMINAL: Partial<Record<RunStatus, true>> = { done: true, failed: true, stopped: true };

export function ConversationsScreen({ client, t }: ScreenProps) {
  const load = useCallback(async () => {
    const [conversations, agents] = await Promise.all([client.listConversations(), client.listAgents()]);
    return { conversations, agents };
  }, [client]);
  const { data, error, refresh } = usePoll(load, 5_000);
  const [selected, setSelected] = useState<string | null>(null);

  const conversations = data?.conversations ?? [];
  const agents = data?.agents ?? [];

  function created(id: string) {
    refresh();
    setSelected(id);
  }

  return (
    <section className="screen split">
      <aside className="card">
        <NewConversation client={client} t={t} agents={agents} onCreated={created} />
        {conversations.length === 0 ? (
          <p className="muted">{t('conversationEmptyList')}</p>
        ) : (
          <ul className="list">
            {conversations.map((conversation: Conversation) => (
              <li key={conversation.id}>
                <button
                  type="button"
                  className={conversation.id === selected ? 'nav-item active' : 'nav-item'}
                  onClick={() => setSelected(conversation.id)}
                >
                  {conversation.title}
                </button>
              </li>
            ))}
          </ul>
        )}
        {error && <p className="error">{error}</p>}
      </aside>
      <div className="card conversation">
        {selected === null ? (
          <div className="empty">
            <h2>{t('conversationEmptyTitle')}</h2>
            <p className="muted">{t('conversationEmptyBody')}</p>
          </div>
        ) : (
          <ConversationView key={selected} client={client} t={t} conversationId={selected} />
        )}
      </div>
    </section>
  );
}

function NewConversation({
  client,
  t,
  agents,
  onCreated,
}: ScreenProps & { agents: Agent[]; onCreated: (id: string) => void }) {
  const [agentId, setAgentId] = useState('');
  const [title, setTitle] = useState('');
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    if (agentId === '' && agents[0]) {
      setAgentId(agents[0].id);
    }
  }, [agentId, agents]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);
    try {
      const conversation = await client.createConversation(agentId, title || undefined);
      setTitle('');
      onCreated(conversation.id);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : String(err));
    }
  }

  if (agents.length === 0) {
    return <p className="muted">{t('conversationNoAgent')}</p>;
  }

  return (
    <form className="form" onSubmit={(event) => void submit(event)}>
      <label>
        {t('conversationAgent')}
        <select value={agentId} onChange={(event) => setAgentId(event.target.value)} required>
          <option value="" disabled>
            —
          </option>
          {agents.map((agent) => (
            <option key={agent.id} value={agent.id}>
              {agent.name}
            </option>
          ))}
        </select>
      </label>
      <label>
        {t('conversationPlaceholder')}
        <input value={title} onChange={(event) => setTitle(event.target.value)} />
      </label>
      {formError && <p className="error">{formError}</p>}
      <button type="submit">{t('actionNewConversation')}</button>
    </form>
  );
}

function ConversationView({ client, t, conversationId }: ScreenProps & { conversationId: string }) {
  const load = useCallback(() => client.listMessages(conversationId), [client, conversationId]);
  const { data, error } = usePoll(load, 2_000);
  const [draft, setDraft] = useState('');
  const [runId, setRunId] = useState<string | null>(null);
  const [sendError, setSendError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);
  const pendingSend = useRef<{ body: string; key: string } | null>(null);

  async function send(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const body = draft.trim();
    if (body === '' || sending) {
      return;
    }
    setSendError(null);
    const key = pendingSend.current?.body === body
      ? pendingSend.current.key
      : crypto.randomUUID();
    pendingSend.current = { body, key };
    setSending(true);
    try {
      const sent = await client.sendMessage(conversationId, body, key);
      pendingSend.current = null;
      setDraft('');
      setRunId(sent.run.id);
    } catch (err) {
      setSendError(err instanceof Error ? err.message : String(err));
    } finally {
      setSending(false);
    }
  }

  const messages: Message[] = data ?? [];
  return (
    <>
      <div className="messages">
        {messages.map((message) => (
          <div key={message.id} className={`message ${message.author_kind}`}>
            {message.body}
          </div>
        ))}
        {error && <p className="error">{error}</p>}
      </div>
      {runId !== null && <RunWatcher key={runId} client={client} t={t} runId={runId} />}
      <form className="composer" onSubmit={(event) => void send(event)}>
        <textarea
          disabled={sending}
          value={draft}
          placeholder={t('conversationPlaceholder')}
          onChange={(event) => {
            const next = event.target.value;
            if (pendingSend.current !== null && pendingSend.current.body !== next.trim()) {
              pendingSend.current = null;
            }
            setDraft(next);
          }}
          rows={2}
        />
        <button type="submit" disabled={sending || draft.trim() === ''}>
          {t('actionSend')}
        </button>
      </form>
      {sendError && <p className="error">{sendError}</p>}
    </>
  );
}

function RunWatcher({ client, t, runId }: ScreenProps & { runId: string }) {
  const load = useCallback(async () => {
    const detail = await client.getRun(runId);
    const approvals = detail.status === 'waiting_approval'
      ? (await client.listApprovals('pending')).filter((approval) => approval.run_id === runId)
      : [];
    return { detail, approvals };
  }, [client, runId]);
  const { data, error, refresh } = usePoll(load, 1_500);
  const [stopError, setStopError] = useState<string | null>(null);
  const [approvalError, setApprovalError] = useState<string | null>(null);
  const [deciding, setDeciding] = useState(false);
  const run = data?.detail;
  const status = run?.status;
  const terminal = status !== undefined && TERMINAL[status] === true;

  async function decide(approval: Approval, decision: 'approve' | 'deny', remember: boolean) {
    setApprovalError(null);
    setDeciding(true);
    try {
      await client.decideApproval(approval.id, decision, remember);
      refresh();
    } catch (err) {
      setApprovalError(err instanceof Error ? err.message : String(err));
    } finally {
      setDeciding(false);
    }
  }

  return (
    <>
      <div className="run">
        <span>{status === undefined ? t('commonLoading') : t(RUN_STATUS_KEY[status])}</span>
        {run?.error_code && <span className="muted"> {run.error_code}</span>}
        {error && <span className="error">{error}</span>}
        {stopError && <span className="error">{stopError}</span>}
        {!terminal && (
          <button type="button" className="danger" onClick={() => {
            setStopError(null);
            void client.stopRun(runId).then(refresh).catch((err) =>
              setStopError(err instanceof Error ? err.message : String(err)),
            );
          }}>
            {t('actionStop')}
          </button>
        )}
      </div>
      {data?.approvals.length ? (
        <div className="card">
          {approvalError && <p className="error">{approvalError}</p>}
          <ul className="list">
            {data.approvals.map((approval) => (
              <li key={approval.id} className="approval">
                <div>
                  <strong>{approval.action}</strong>
                  <div className="muted">
                    {t('approvalTarget')}: {approval.target}
                  </div>
                </div>
                <div className="actions">
                  <button type="button" disabled={deciding} onClick={() => void decide(approval, 'approve', false)}>
                    {t('actionApprove')}
                  </button>
                  <button type="button" disabled={deciding} onClick={() => void decide(approval, 'approve', true)}>
                    {t('actionAlwaysAllow')}
                  </button>
                  <button type="button" className="danger" disabled={deciding} onClick={() => void decide(approval, 'deny', false)}>
                    {t('actionDeny')}
                  </button>
                </div>
              </li>
            ))}
          </ul>
        </div>
      ) : approvalError ? (
        <p className="error">{approvalError}</p>
      ) : null}
    </>
  );
}
