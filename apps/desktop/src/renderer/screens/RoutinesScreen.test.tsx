import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { anAgent, aRoutine, fakeClient, tr } from '../test/fakes';
import { RoutinesScreen } from './RoutinesScreen';

const t = tr('en');

describe('RoutinesScreen', () => {
  it('mostra vazio e pede agente antes do formulário', async () => {
    render(<RoutinesScreen client={fakeClient({ listRoutines: async () => [], listAgents: async () => [] })} t={t} />);
    expect(await screen.findByText(t('routineEmpty'))).toBeInTheDocument();
    expect(screen.getByText(t('agentEmpty'))).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: t('routineCreate') })).not.toBeInTheDocument();
  });

  it('cria uma rotina por intervalo com o formulário', async () => {
    const user = userEvent.setup();
    let routines = [] as ReturnType<typeof aRoutine>[];
    const client = fakeClient({
      listRoutines: async () => routines,
      listAgents: async () => [anAgent()],
      createRoutine: async (input) => {
        routines = [aRoutine({ name: input.name, schedule: input.schedule })];
        return routines[0];
      },
    });
    render(<RoutinesScreen client={client} t={t} />);

    await user.type(await screen.findByLabelText(t('routineName')), 'Resumo diário');
    await user.type(screen.getByLabelText(t('routinePrompt')), 'Resuma o dia');
    await user.selectOptions(screen.getByLabelText(t('routineTrigger')), 'interval');
    await user.type(screen.getByLabelText(t('routineSchedule'), { selector: 'input' }), '1h');
    await user.selectOptions(screen.getByLabelText(t('routineNotify')), 'failure');
    await user.click(screen.getByRole('button', { name: t('routineCreate') }));

    expect(client.createRoutine).toHaveBeenCalledWith({
      agent_id: 'agent-1',
      name: 'Resumo diário',
      prompt: 'Resuma o dia',
      trigger_kind: 'interval',
      schedule: '1h',
      notify_on: 'failure',
    });
    const item = (await screen.findByText('Resumo diário', { selector: 'strong' })).closest('li') as HTMLElement;
    expect(within(item).getByText(/1h/)).toBeInTheDocument();
    expect(screen.getByLabelText(t('routineName'))).toHaveValue('');
  });

  it('mostra o erro de validação do daemon e mantém o que foi digitado', async () => {
    const user = userEvent.setup();
    const client = fakeClient({
      listRoutines: async () => [],
      listAgents: async () => [anAgent()],
      createRoutine: async () => Promise.reject(new Error('interval must be a duration of at least 1m')),
    });
    render(<RoutinesScreen client={client} t={t} />);
    await user.type(await screen.findByLabelText(t('routineName')), 'Rápida');
    await user.type(screen.getByLabelText(t('routinePrompt')), 'Oi');
    await user.selectOptions(screen.getByLabelText(t('routineTrigger')), 'interval');
    await user.type(screen.getByLabelText(t('routineSchedule'), { selector: 'input' }), '10s');
    await user.click(screen.getByRole('button', { name: t('routineCreate') }));
    expect(await screen.findByText('interval must be a duration of at least 1m')).toBeInTheDocument();
    expect(screen.getByLabelText(t('routineName'))).toHaveValue('Rápida');
  });

  it('pausa e retoma uma rotina', async () => {
    const user = userEvent.setup();
    let routine = aRoutine();
    const client = fakeClient({
      listRoutines: async () => [routine],
      listAgents: async () => [anAgent()],
      updateRoutine: async (_id, patch) => {
        routine = aRoutine({ paused: patch.paused });
        return routine;
      },
    });
    render(<RoutinesScreen client={client} t={t} />);
    await user.click(await screen.findByRole('button', { name: t('routinePause') }));
    expect(client.updateRoutine).toHaveBeenCalledWith('rt-1', { paused: true });
    await user.click(await screen.findByRole('button', { name: t('routineResume') }));
    expect(client.updateRoutine).toHaveBeenLastCalledWith('rt-1', { paused: false });
    expect(await screen.findByRole('button', { name: t('routinePause') })).toBeInTheDocument();
  });

  it('mostra o erro ao pausar', async () => {
    const user = userEvent.setup();
    const client = fakeClient({
      listRoutines: async () => [aRoutine()],
      listAgents: async () => [anAgent()],
      updateRoutine: async () => Promise.reject(new Error('routine not found')),
    });
    render(<RoutinesScreen client={client} t={t} />);
    await user.click(await screen.findByRole('button', { name: t('routinePause') }));
    expect(await screen.findByText('routine not found')).toBeInTheDocument();
  });
});
