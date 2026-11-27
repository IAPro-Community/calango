import type { CalangoClient } from '@calango/sdk-ts';
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { anAgent, anAccount, anApproval, fakeBridge, fakeClient, tr } from './test/fakes';

const sdk = vi.hoisted(() => ({ createClient: vi.fn() }));
vi.mock('@calango/sdk-ts', () => sdk);

const { App } = await import('./App');

const en = tr('en');
const pt = tr('pt-BR');

// renderApp waits for the first getState reply, as IPC delivers it before later state pushes.
async function renderApp() {
  render(<App />);
  await act(async () => {});
}

function connectedClient(): CalangoClient {
  return fakeClient({ listAgents: async () => [anAgent()], listProviderAccounts: async () => [anAccount()] });
}

describe('App', () => {
  beforeEach(() => {
    sdk.createClient.mockImplementation(connectedClient);
  });

  it('mostra carregando até o daemon responder', async () => {
    await renderApp();
    expect(screen.getByRole('heading', { name: en('commonLoading') })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: en('appName') })).toBeInTheDocument();
  });

  it('reconecta com um token novo depois que o daemon reinicia', async () => {
    await renderApp();
    act(() => fakeBridge.emitState({ status: 'running' }, { baseUrl: 'http://127.0.0.1:7317', token: 'token-1' }));
    expect(await screen.findByText('Ana')).toBeInTheDocument();

    act(() => fakeBridge.emitState({ status: 'restarting', detail: 'exited (code 1); retrying in 250 ms' }));
    expect(await screen.findByRole('heading', { name: en('daemonRetrying') })).toBeInTheDocument();
    expect(screen.getByText('exited (code 1); retrying in 250 ms')).toBeInTheDocument();
    expect(screen.queryByText('Ana')).not.toBeInTheDocument();

    act(() => fakeBridge.emitState({ status: 'running' }, { baseUrl: 'http://127.0.0.1:7317', token: 'token-2' }));
    expect(await screen.findByText('Ana')).toBeInTheDocument();
    expect(sdk.createClient).toHaveBeenLastCalledWith({ baseUrl: 'http://127.0.0.1:7317', token: 'token-2' });
    expect(fakeBridge.api.getConnection).toHaveBeenCalledTimes(2);
  });

  it('mostra o daemon indisponível com o motivo', async () => {
    await renderApp();
    act(() => fakeBridge.emitState({ status: 'stopped', detail: 'spawn: ENOENT' }));
    expect(await screen.findByRole('heading', { name: en('daemonUnavailable') })).toBeInTheDocument();
    expect(screen.getByText('spawn: ENOENT')).toBeInTheDocument();
  });

  it('navega pelas seções com teclado e marca a atual', async () => {
    const user = userEvent.setup();
    await renderApp();
    act(() => fakeBridge.emitState({ status: 'running' }, { baseUrl: 'http://127.0.0.1:7317', token: 't' }));
    await screen.findByText('Ana');

    const agents = screen.getByRole('button', { name: en('navAgents') });
    expect(agents).toHaveAttribute('aria-current', 'page');
    // The menu follows the Figma sidebar: conversations come first.
    await user.tab();
    const conversations = screen.getByRole('button', { name: en('navConversations') });
    expect(conversations).toHaveFocus();

    sdk.createClient.mockImplementation(() =>
      fakeClient({ listConversations: async () => [], listAgents: async () => [anAgent()] }),
    );
    await user.keyboard('{Enter}');
    expect(conversations).toHaveAttribute('aria-current', 'page');
    expect(agents).not.toHaveAttribute('aria-current');
  });

  it('conta as aprovações pendentes no menu sem mudar o nome do item', async () => {
    const listApprovals = vi.fn(async () => [anApproval(), anApproval({ id: 'apr-2' })]);
    sdk.createClient.mockImplementation(() =>
      fakeClient({
        listAgents: async () => [anAgent()],
        listProviderAccounts: async () => [anAccount()],
        listApprovals,
      }),
    );
    await renderApp();
    act(() => fakeBridge.emitState({ status: 'running' }, { baseUrl: 'http://127.0.0.1:7317', token: 't' }));

    const approvals = screen.getByRole('button', { name: en('navApprovals') });
    await waitFor(() => expect(approvals).toHaveAccessibleDescription(`2 ${en('navPending')}`));
    expect(listApprovals).toHaveBeenCalledWith('pending');
  });

  it('abre conversas pelo atalho de conversa rápida', async () => {
    const client = fakeClient({
      listAgents: async () => [anAgent()],
      listProviderAccounts: async () => [],
      listConversations: async () => [],
    });
    sdk.createClient.mockImplementation(() => client);
    await renderApp();
    act(() => fakeBridge.emitState({ status: 'running' }, { baseUrl: 'http://127.0.0.1:7317', token: 't' }));
    await screen.findByText('Ana');
    act(() => fakeBridge.quickConversation());
    expect(screen.getByRole('button', { name: en('navConversations') })).toHaveAttribute('aria-current', 'page');
    await waitFor(() => expect(client.listConversations).toHaveBeenCalled());
  });

  it('usa português quando o sistema não está em inglês nem espanhol', () => {
    vi.spyOn(navigator, 'language', 'get').mockReturnValue('pt-BR');
    render(<App />);
    expect(screen.getByRole('button', { name: pt('navAgents') })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: pt('commonLoading') })).toBeInTheDocument();
  });

  it('usa espanhol quando o sistema está em espanhol', () => {
    vi.spyOn(navigator, 'language', 'get').mockReturnValue('es-AR');
    render(<App />);
    expect(screen.getByRole('button', { name: tr('es')('navAgents') })).toBeInTheDocument();
  });
});
