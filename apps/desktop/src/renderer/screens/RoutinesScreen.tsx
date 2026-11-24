import { useCallback, useEffect, useState, type FormEvent } from 'react';
import type { Agent, Routine } from '@calango/sdk-ts';
import { usePoll } from '../usePoll';
import type { ScreenProps } from './props';

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
    <section className="screen">
      <header className="screen-header">
        <h1>{t('navRoutines')}</h1>
      </header>
      {error && <p className="error">{error}</p>}
      {actionError && <p className="error">{actionError}</p>}
      <div className="grid">
        <div className="card">
          <h2>{t('routineCreate')}</h2>
          <RoutineForm client={client} t={t} agents={agents} onCreated={refresh} />
        </div>
        <div className="card">
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
                  <button type="button" onClick={() => void togglePaused(routine)}>
                    {routine.paused ? t('routineResume') : t('routinePause')}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
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
    <form className="form" onSubmit={(event) => void submit(event)}>
      <label>
        {t('routineAgent')}
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
        {t('routineName')}
        <input value={name} onChange={(event) => setName(event.target.value)} required />
      </label>
      <label>
        {t('routinePrompt')}
        <textarea value={prompt} onChange={(event) => setPrompt(event.target.value)} rows={3} required />
      </label>
      <label>
        {t('routineTrigger')}
        <select value={triggerKind} onChange={(event) => setTriggerKind(event.target.value as 'cron' | 'interval')}>
          <option value="cron">{t('triggerCron')}</option>
          <option value="interval">{t('triggerInterval')}</option>
        </select>
      </label>
      <label>
        {t('routineSchedule')}
        <input value={schedule} onChange={(event) => setSchedule(event.target.value)} required />
      </label>
      <label>
        {t('routineNotify')}
        <select value={notifyOn} onChange={(event) => setNotifyOn(event.target.value as 'always' | 'news' | 'failure')}>
          <option value="always">{t('notifyAlways')}</option>
          <option value="news">{t('notifyNews')}</option>
          <option value="failure">{t('notifyFailure')}</option>
        </select>
      </label>
      {formError && <p className="error">{formError}</p>}
      <button type="submit">{t('routineCreate')}</button>
    </form>
  );
}
