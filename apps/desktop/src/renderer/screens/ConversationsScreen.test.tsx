import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { aConversation, aMessage, anAgent, anApproval, aRun, fakeClient, pending, tr } from '../test/fakes';
import { ConversationsScreen } from './ConversationsScreen';

const t = tr('en');

function setup(overrides: Parameters<typeof fakeClient>[0] = {}) {
  const client = fakeClient({
    listConversations: async () => [aConversation()],
    listAgents: async () => [anAgent()],
    listMessages: async () => [aMessage({ body: 'Olá' }), aMessage({ id: 'msg-2', author_kind: 'agent', body: 'Oi!' })],
    ...overrides,
  });
  render(<ConversationsScreen client={client} t={t} />);
  return client;
}

describe('ConversationsScreen', () => {
  it('mostra o estado vazio antes de escolher uma conversa', async () => {
    setup({ listConversations: async () => [] });
    expect(await screen.findByText(t('conversationEmptyList'))).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: t('conversationEmptyTitle') })).toBeInTheDocument();
  });

  it('pede um agente antes de criar conversa', async () => {
    setup({ listConversations: async () => [], listAgents: async () => [] });
    expect(await screen.findByText(t('conversationNoAgent'))).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: t('actionNewConversation') })).not.toBeInTheDocument();
  });

  it('mostra o erro de carregamento da lista', async () => {
    setup({ listConversations: async () => Promise.reject(new Error('daemon fora do ar')) });
    expect(await screen.findByText('daemon fora do ar')).toBeInTheDocument();
  });

  it('cria uma conversa e abre ela', async () => {
    const user = userEvent.setup();
    const client = setup({
      listConversations: async () => [],
      createConversation: async () => aConversation({ id: 'conv-9', title: 'Nova' }),
      listMessages: async () => [],
    });
    await user.type(await screen.findByLabelText(t('conversationPlaceholder')), 'Nova');
    await user.click(screen.getByRole('button', { name: t('actionNewConversation') }));
    expect(client.createConversation).toHaveBeenCalledWith('agent-1', 'Nova');
    await waitFor(() => expect(client.listMessages).toHaveBeenCalledWith('conv-9'));
  });

  it('lista mensagens e envia com idempotência pelo teclado', async () => {
    const user = userEvent.setup();
    const client = setup({
      sendMessage: async () => ({ message: aMessage({ id: 'msg-3' }), run: aRun() }),
      getRun: async () => aRun({ status: 'done' }),
    });
    await user.click(await screen.findByRole('button', { name: 'Planejamento' }));
    expect(await screen.findByText('Oi!')).toBeInTheDocument();

    const send = screen.getByRole('button', { name: t('actionSend') });
    expect(send).toBeDisabled();
    const composer = screen.getByPlaceholderText(t('conversationPlaceholder'));
    await user.type(composer, '  Tudo bem?  ');
    expect(send).toBeEnabled();
    await user.tab();
    expect(send).toHaveFocus();
    await user.keyboard('{Enter}');

    expect(client.sendMessage).toHaveBeenCalledWith('conv-1', 'Tudo bem?', expect.any(String));
    await waitFor(() => expect(composer).toHaveValue(''));
    expect(await screen.findByText(t('runDone'))).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: t('actionStop') })).not.toBeInTheDocument();
  });

  it('reenvia com a mesma chave depois de uma falha', async () => {
    const user = userEvent.setup();
    const client = setup({ sendMessage: async () => Promise.reject(new Error('rede caiu')) });
    await user.click(await screen.findByRole('button', { name: 'Planejamento' }));
    await user.type(screen.getByPlaceholderText(t('conversationPlaceholder')), 'Oi');
    await user.click(screen.getByRole('button', { name: t('actionSend') }));
    expect(await screen.findByText('rede caiu')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: t('actionSend') }));

    expect(client.sendMessage).toHaveBeenCalledTimes(2);
    const [first, second] = client.sendMessage.mock.calls;
    expect(second[2]).toBe(first[2]);
    expect(screen.getByPlaceholderText(t('conversationPlaceholder'))).toHaveValue('Oi');
  });

  it('para uma execução em andamento', async () => {
    const user = userEvent.setup();
    let status: 'running' | 'stopped' = 'running';
    const client = setup({
      sendMessage: async () => ({ message: aMessage({ id: 'msg-3' }), run: aRun() }),
      getRun: async () => aRun({ status }),
      stopRun: async () => {
        status = 'stopped';
      },
    });
    await user.click(await screen.findByRole('button', { name: 'Planejamento' }));
    await user.type(screen.getByPlaceholderText(t('conversationPlaceholder')), 'Demore');
    await user.click(screen.getByRole('button', { name: t('actionSend') }));

    await user.click(await screen.findByRole('button', { name: t('actionStop') }));
    expect(client.stopRun).toHaveBeenCalledWith('run-1');
    expect(await screen.findByText(t('runStopped'))).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: t('actionStop') })).not.toBeInTheDocument();
  });

  it('decide a aprovação pendente da execução na própria conversa', async () => {
    const user = userEvent.setup();
    let status: 'waiting_approval' | 'done' = 'waiting_approval';
    const client = setup({
      sendMessage: async () => ({ message: aMessage({ id: 'msg-3' }), run: aRun() }),
      getRun: async () => aRun({ status }),
      listApprovals: async () => [anApproval(), anApproval({ id: 'ap-other', run_id: 'run-other' })],
      decideApproval: async () => {
        status = 'done';
        return anApproval({ status: 'approved' });
      },
    });
    await user.click(await screen.findByRole('button', { name: 'Planejamento' }));
    await user.type(screen.getByPlaceholderText(t('conversationPlaceholder')), 'Salve a nota');
    await user.click(screen.getByRole('button', { name: t('actionSend') }));

    expect(await screen.findByText(t('runWaitingApproval'))).toBeInTheDocument();
    const items = await screen.findAllByRole('listitem');
    const approval = items.find((item) => item.classList.contains('approval'));
    expect(approval).toBeDefined();
    expect(screen.getAllByText(/notas\/create_note/)).toHaveLength(1);
    await user.click(within(approval as HTMLElement).getByRole('button', { name: t('actionApprove') }));

    expect(client.decideApproval).toHaveBeenCalledWith('ap-1', 'approve', false);
    expect(await screen.findByText(t('runDone'))).toBeInTheDocument();
  });

  it('desabilita o compositor enquanto envia', async () => {
    const user = userEvent.setup();
    setup({ sendMessage: () => pending() });
    await user.click(await screen.findByRole('button', { name: 'Planejamento' }));
    const composer = screen.getByPlaceholderText(t('conversationPlaceholder'));
    await user.type(composer, 'Oi');
    await user.click(screen.getByRole('button', { name: t('actionSend') }));
    expect(composer).toBeDisabled();
    expect(screen.getByRole('button', { name: t('actionSend') })).toBeDisabled();
  });
});
