import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { anAccount, anAgent, fakeClient, tr } from '../test/fakes';
import { AgentsScreen } from './AgentsScreen';

const t = tr('en');

function accountForm() {
  return screen.getByLabelText(t('accountKind')).closest('form') as HTMLFormElement;
}

describe('AgentsScreen', () => {
  it('pede uma conta antes de criar agente e mostra vazio', async () => {
    render(
      <AgentsScreen client={fakeClient({ listAgents: async () => [], listProviderAccounts: async () => [] })} t={t} />,
    );
    expect(await screen.findByText(t('agentNeedsAccount'))).toBeInTheDocument();
    expect(screen.getByText(t('agentEmpty'))).toBeInTheDocument();
  });

  it('cria conta Codex sem pedir segredo', async () => {
    const user = userEvent.setup();
    let accounts: ReturnType<typeof anAccount>[] = [];
    const client = fakeClient({
      listAgents: async () => [],
      listProviderAccounts: async () => accounts,
      createProviderAccount: async () => {
        accounts = [anAccount()];
        return accounts[0];
      },
    });
    render(<AgentsScreen client={client} t={t} />);
    await screen.findByText(t('agentNeedsAccount'));
    expect(within(accountForm()).queryByLabelText(t('accountApiKey'))).not.toBeInTheDocument();
    await user.click(within(accountForm()).getByRole('button', { name: t('accountCreate') }));
    expect(client.createProviderAccount).toHaveBeenCalledWith({ kind: 'codex', label: 'default' });
    expect(client.setVaultSecret).not.toHaveBeenCalled();
    expect(await screen.findByLabelText(t('agentName'))).toBeInTheDocument();
  });

  it('guarda a chave Anthropic no cofre e limpa o campo', async () => {
    const user = userEvent.setup();
    const client = fakeClient({
      listAgents: async () => [],
      listProviderAccounts: async () => [],
      createProviderAccount: async () => anAccount({ kind: 'anthropic', key: 'anthropic/work', label: 'work', secret_item: 'anthropic-work-api-key' }),
      setVaultSecret: async () => {},
    });
    render(<AgentsScreen client={client} t={t} />);
    await screen.findByText(t('agentNeedsAccount'));
    const form = within(accountForm());
    await user.selectOptions(form.getByLabelText(t('accountKind')), 'anthropic');
    await user.clear(form.getByLabelText(t('accountLabel')));
    await user.type(form.getByLabelText(t('accountLabel')), 'work');
    const key = form.getByLabelText(t('accountApiKey'));
    expect(key).toHaveAttribute('type', 'password');
    expect(key).toHaveAttribute('autocomplete', 'off');
    await user.type(key, 'sk-ant-teste');
    await user.click(form.getByRole('button', { name: t('accountCreate') }));

    expect(client.createProviderAccount).toHaveBeenCalledWith({ kind: 'anthropic', label: 'work' });
    expect(client.setVaultSecret).toHaveBeenCalledWith('anthropic-work-api-key', 'sk-ant-teste');
    expect(key).toHaveValue('');
  });

  it('lista somente os providers habilitados pelo daemon', async () => {
    const client = fakeClient({
      listAgents: async () => [],
      listProviderAccounts: async () => [],
      listProviderKinds: async () => [{ kind: 'anthropic', display_name: 'Anthropic', auth_kind: 'api_key' }],
    });
    render(<AgentsScreen client={client} t={t} />);
    expect(await screen.findByRole('option', { name: 'Anthropic' })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Codex' })).not.toBeInTheDocument();
    expect(await screen.findByLabelText(t('accountApiKey'))).toBeInTheDocument();
  });

  it('mostra o erro da conta no formulário', async () => {
    const user = userEvent.setup();
    const client = fakeClient({
      listAgents: async () => [],
      listProviderAccounts: async () => [],
      createProviderAccount: async () => Promise.reject(new Error('account already exists')),
    });
    render(<AgentsScreen client={client} t={t} />);
    await screen.findByText(t('agentNeedsAccount'));
    await user.click(within(accountForm()).getByRole('button', { name: t('accountCreate') }));
    expect(await within(accountForm()).findByText('account already exists')).toBeInTheDocument();
  });

  it('cria agente com a conta escolhida e lista o resultado', async () => {
    const user = userEvent.setup();
    let agents: ReturnType<typeof anAgent>[] = [];
    const client = fakeClient({
      listAgents: async () => agents,
      listProviderAccounts: async () => [anAccount()],
      createAgent: async (input) => {
        agents = [anAgent({ name: input.name, role: input.role })];
        return agents[0];
      },
    });
    render(<AgentsScreen client={client} t={t} />);
    await user.type(await screen.findByLabelText(t('agentName')), 'Bia');
    await user.type(screen.getByLabelText(t('agentRole')), 'revisão');
    await user.type(screen.getByLabelText(t('agentModel'), { exact: false }), 'gpt-fake');
    await user.selectOptions(screen.getByLabelText(t('agentApprovalLevel')), 'always');
    await user.click(screen.getByRole('button', { name: t('agentCreate') }));

    expect(client.createAgent).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Bia',
        role: 'revisão',
        providers: [{ account_id: 'acc-1', model: 'gpt-fake' }],
        approval_level: 'always',
        maestro_enabled: false,
      }),
    );
    expect(await screen.findByText('Bia')).toBeInTheDocument();
    expect(screen.getByText(`revisão · ${t('agentStatusActive')}`)).toBeInTheDocument();
    expect(screen.getByLabelText(t('agentName'))).toHaveValue('');
  });

  it('liga o Maestro de um agente Codex', async () => {
    const user = userEvent.setup();
    const client = fakeClient({
      listAgents: async () => [anAgent()],
      listProviderAccounts: async () => [anAccount()],
      updateAgent: async (_id, input) => anAgent({ maestro_enabled: input.maestro_enabled }),
    });
    render(<AgentsScreen client={client} t={t} />);
    const item = (await screen.findByText('Ana')).closest('li') as HTMLElement;
    await user.click(within(item).getByRole('button', { name: t('agentMaestroLink') }));
    expect(client.updateAgent).toHaveBeenCalledWith('agent-1', expect.objectContaining({ maestro_enabled: true }));
  });

  it('pausa todos e mostra a falha', async () => {
    const user = userEvent.setup();
    const client = fakeClient({
      listAgents: async () => [anAgent({ paused: true })],
      listProviderAccounts: async () => [anAccount()],
      pauseAll: async () => Promise.reject(new Error('store locked')),
    });
    render(<AgentsScreen client={client} t={t} />);
    expect(await screen.findByText(`pesquisa · ${t('agentStatusPaused')}`)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: t('agentPauseAll') }));
    expect(await screen.findByText('store locked')).toBeInTheDocument();
  });

  it('mostra o erro de carregamento', async () => {
    render(
      <AgentsScreen
        client={fakeClient({
          listAgents: async () => Promise.reject(new Error('offline')),
          listProviderAccounts: async () => [],
        })}
        t={t}
      />,
    );
    expect(await screen.findByText('offline')).toBeInTheDocument();
  });
});
