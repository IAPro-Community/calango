import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react';
import type { Agent, Approval, Conversation, Message, Run } from '@calango/sdk-ts';
import { ApprovalCard, Badge, type BadgeProps, Button, Card, cn, Field, Input, PageHeader, Select, Textarea } from '@';
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

const RUN_TONE: Record<RunStatus, NonNullable<BadgeProps['tone']>> = {
  queued: 'neutral',
  running: 'warn',
  waiting_approval: 'postit',
  done: 'ok',
  failed: 'danger',
  stopped: 'neutral',
};

const TERMINAL: Partial<Record<RunStatus, true>> = { done: true, failed: true, stopped: true };

const AUTHOR_KEY: Record<Message['author_kind'], MessageKey> = {
  person: 'conversationAuthorPerson',
  agent: 'conversationAuthorAgent',
  system: 'conversationAuthorSystem',
};

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
    <section className="flex h-full min-h-0 flex-col gap-md">
      <PageHeader title={t('navConversations')} />
      <div className="grid min-h-0 flex-1 grid-cols-[280px_minmax(0,1fr)] gap-md">
        <Card className="self-start">
          <NewConversation client={client} t={t} agents={agents} onCreated={created} />
          {conversations.length === 0 ? (
            <p className="muted">{t('conversationEmptyList')}</p>
          ) : (
            <ul className="m-0 flex list-none flex-col gap-xs p-0">
              {conversations.map((conversation: Conversation) => {
                const active = conversation.id === selected;
                return (
                  <li key={conversation.id}>
                    <button
                      type="button"
                      className={cn(
                        'flex w-full cursor-pointer items-center gap-xs rounded-md border px-sm py-sm text-left font-body text-[14px] leading-5 font-semibold text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-postit',
                        active ? 'border-leaf bg-ok-soft' : 'border-line bg-paper hover:brightness-110',
                      )}
                      aria-current={active ? 'true' : undefined}
                      onClick={() => setSelected(conversation.id)}
                    >
                      <span
                        aria-hidden="true"
                        className={cn('size-2 shrink-0 rounded-[2px]', active ? 'bg-leaf' : 'bg-clay')}
                      />
                      <span className="truncate">{conversation.title}</span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
          {error && (
            <p className="error" role="alert">
              {error}
            </p>
          )}
        </Card>
        <Card className="min-h-[480px]">
          {selected === null ? (
            <div className="m-auto flex max-w-sm flex-col items-center gap-xs text-center">
              <h2 className="m-0 font-display text-[18px] font-semibold text-ink">{t('conversationEmptyTitle')}</h2>
              <p className="muted">{t('conversationEmptyBody')}</p>
            </div>
          ) : (
            <ConversationView key={selected} client={client} t={t} conversationId={selected} />
          )}
        </Card>
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
    <form className="flex flex-col gap-sm" onSubmit={(event) => void submit(event)}>
      <Field label={t('conversationAgent')}>
        <Select value={agentId} onChange={(event) => setAgentId(event.target.value)} required>
          <option value="" disabled>
            —
          </option>
          {agents.map((agent) => (
            <option key={agent.id} value={agent.id}>
              {agent.name}
            </option>
          ))}
        </Select>
      </Field>
      <Field label={t('conversationTitleLabel')}>
        <Input value={title} onChange={(event) => setTitle(event.target.value)} />
      </Field>
      {formError && (
        <p className="error" role="alert">
          {formError}
        </p>
      )}
      <Button type="submit" variant="postit">
        {t('actionNewConversation')}
      </Button>
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
    const key = pendingSend.current?.body === body ? pendingSend.current.key : crypto.randomUUID();
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
      <div
        className="flex flex-1 flex-col gap-sm overflow-y-auto"
        role="log"
        aria-label={t('conversationMessagesLabel')}
      >
        {messages.map((message) =>
          message.author_kind === 'person' ? (
            <div
              key={message.id}
              className="message person max-w-[75%] self-end rounded-lg bg-btn px-md py-sm font-body text-[14px] leading-[21px] whitespace-pre-wrap text-on-btn"
            >
              <span className="sr-only">{t(AUTHOR_KEY[message.author_kind])}: </span>
              {message.body}
            </div>
          ) : (
            <div
              key={message.id}
              className={`message ${message.author_kind} flex max-w-[85%] flex-col gap-2xs self-start rounded-lg border border-line bg-paper px-md py-sm`}
            >
              <span className="font-mono text-[11px] leading-[14px] font-bold tracking-[0.08em] text-moss-text uppercase">
                {t(AUTHOR_KEY[message.author_kind])}
                <span className="sr-only">: </span>
              </span>
              <span className="font-body text-[14px] leading-[21px] whitespace-pre-wrap text-ink">{message.body}</span>
            </div>
          ),
        )}
        {error && (
          <p className="error" role="alert">
            {error}
          </p>
        )}
      </div>
      {runId !== null && <RunWatcher key={runId} client={client} t={t} runId={runId} />}
      <form
        className="composer flex items-end gap-xs rounded-lg border border-line bg-paper p-sm"
        onSubmit={(event) => void send(event)}
      >
        <Textarea
          className="min-h-11 flex-1 border-transparent bg-transparent"
          aria-label={t('conversationMessageLabel')}
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
        <Button type="submit" disabled={sending || draft.trim() === ''}>
          {t('actionSend')}
        </Button>
      </form>
      {sendError && (
        <p className="error" role="alert">
          {sendError}
        </p>
      )}
    </>
  );
}

function RunWatcher({ client, t, runId }: ScreenProps & { runId: string }) {
  const load = useCallback(async () => {
    const detail = await client.getRun(runId);
    const approvals =
      detail.status === 'waiting_approval'
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
      <div className="run flex flex-wrap items-center gap-xs">
        <output>
          <Badge tone={status === undefined ? 'neutral' : RUN_TONE[status]}>
            {status === undefined ? t('commonLoading') : t(RUN_STATUS_KEY[status])}
          </Badge>
        </output>
        {run?.error_code && <span className="font-mono text-[12px] text-ink-soft">{run.error_code}</span>}
        {error && (
          <span className="error" role="alert">
            {error}
          </span>
        )}
        {stopError && (
          <span className="error" role="alert">
            {stopError}
          </span>
        )}
        {!terminal && (
          <Button
            variant="danger"
            size="sm"
            onClick={() => {
              setStopError(null);
              void client
                .stopRun(runId)
                .then(refresh)
                .catch((err) => setStopError(err instanceof Error ? err.message : String(err)));
            }}
          >
            {t('actionStop')}
          </Button>
        )}
      </div>
      {approvalError && (
        <p className="error" role="alert">
          {approvalError}
        </p>
      )}
      {data?.approvals.length ? (
        <ul className="m-0 flex max-w-[560px] list-none flex-col gap-sm p-0">
          {data.approvals.map((approval) => (
            <li key={approval.id} className="approval">
              <ApprovalCard
                state="pending"
                eyebrow={t('approvalPendingEyebrow')}
                title={approval.action}
                detail={`${t('approvalTarget')}: ${approval.target}`}
                actions={
                  <>
                    <Button disabled={deciding} onClick={() => void decide(approval, 'approve', false)}>
                      {t('actionApprove')}
                    </Button>
                    <Button variant="ghost" disabled={deciding} onClick={() => void decide(approval, 'approve', true)}>
                      {t('actionAlwaysAllow')}
                    </Button>
                    <Button variant="danger" disabled={deciding} onClick={() => void decide(approval, 'deny', false)}>
                      {t('actionDeny')}
                    </Button>
                  </>
                }
              />
            </li>
          ))}
        </ul>
      ) : null}
    </>
  );
}
