import { useCallback, useEffect, useState, type FormEvent } from 'react';
import type { Agent, Routine } from '@calango/sdk-ts';
import { usePoll } from '../usePoll';
import type { ScreenProps } from './props';
import { Button, Card, Field, Input, PageHeader, Select, Textarea } from '@';

export function RoutinesScreen({ client, t }: ScreenProps) {
  const load = useCallback(async () => {
    const [routines, agents] = await Promise.all([client.listRoutines(), client.listAgents()]);
    return { routines, agents };
  }, [client]);
  const { data, error, refresh } = usePoll(load, 10_000);
  const routines = data?.routines ?? [];
  const agents = data?.agents ?? [];
  const [actionError, setActionError] = useState<string | null>(null);

  async function togglePaused(routine: Routine) {
    setActionError(null);
    try {
      await client.updateRoutine(routine.id, { paused: !routine.paused });
      refresh();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : String(err));
    }
  }

  return (
    <section className="flex flex-col gap-md">
      <PageHeader title={t('navRoutines')} />
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
        <Card title={t('routineCreate')}>
          <RoutineForm client={client} t={t} agents={agents} onCreated={refresh} />
        </Card>
        <Card>
          {routines.length === 0 ? (
            <p className="muted">{t('routineEmpty')}</p>
          ) : (
            <ul className="list">
              {routines.map((routine) => (
                <li key={routine.id}>
                  <div>
                    <strong>{routine.name}</strong>
                    <div className="muted">
                      {t('routineSchedule')}: {routine.schedule} · {t('routineNextRun')}: {routine.next_run_at}
                    </div>
                  </div>
                  <Button variant="ghost" onClick={() => void togglePaused(routine)}>
                    {routine.paused ? t('routineResume') : t('routinePause')}
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>
    </section>
  );
}

function RoutineForm({ client, t, agents, onCreated }: ScreenProps & { agents: Agent[]; onCreated: () => void }) {
  const [agentId, setAgentId] = useState('');
  const [name, setName] = useState('');
  const [prompt, setPrompt] = useState('');
  const [triggerKind, setTriggerKind] = useState<'cron' | 'interval'>('cron');
  const [schedule, setSchedule] = useState('');
  const [notifyOn, setNotifyOn] = useState<'always' | 'news' | 'failure'>('news');
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
      await client.createRoutine({
        agent_id: agentId,
        name,
        prompt,
        trigger_kind: triggerKind,
        schedule,
        notify_on: notifyOn,
      });
      setName('');
      setPrompt('');
      setSchedule('');
      onCreated();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : String(err));
    }
  }

  if (agents.length === 0) {
    return <p className="muted">{t('agentEmpty')}</p>;
  }

  return (
    <form className="flex flex-col gap-sm" onSubmit={(event) => void submit(event)}>
      <Field label={t('routineAgent')}>
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
      <Field label={t('routineName')}>
        <Input value={name} onChange={(event) => setName(event.target.value)} required />
      </Field>
      <Field label={t('routinePrompt')}>
        <Textarea value={prompt} onChange={(event) => setPrompt(event.target.value)} rows={3} required />
      </Field>
      <Field label={t('routineTrigger')}>
        <Select value={triggerKind} onChange={(event) => setTriggerKind(event.target.value as 'cron' | 'interval')}>
          <option value="cron">{t('triggerCron')}</option>
          <option value="interval">{t('triggerInterval')}</option>
        </Select>
      </Field>
      <Field label={t('routineSchedule')}>
        <Input value={schedule} onChange={(event) => setSchedule(event.target.value)} required />
      </Field>
      <Field label={t('routineNotify')}>
        <Select value={notifyOn} onChange={(event) => setNotifyOn(event.target.value as 'always' | 'news' | 'failure')}>
          <option value="always">{t('notifyAlways')}</option>
          <option value="news">{t('notifyNews')}</option>
          <option value="failure">{t('notifyFailure')}</option>
        </Select>
      </Field>
      {formError && (
        <p className="error" role="alert">
          {formError}
        </p>
      )}
      <Button type="submit">{t('routineCreate')}</Button>
    </form>
  );
}
