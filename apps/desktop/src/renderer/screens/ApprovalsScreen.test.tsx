import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { anApproval, fakeClient, pending, tr } from '../test/fakes';
import { ApprovalsScreen } from './ApprovalsScreen';

const t = tr('en');

describe('ApprovalsScreen', () => {
  it('lista só as pendentes e mostra vazio quando não há nenhuma', async () => {
    const client = fakeClient({ listApprovals: async () => [] });
    render(<ApprovalsScreen client={client} t={t} />);
    expect(await screen.findByText(t('approvalEmpty'))).toBeInTheDocument();
    expect(client.listApprovals).toHaveBeenCalledWith('pending');
  });

  it('mostra ação e alvo de cada pedido', async () => {
    render(<ApprovalsScreen client={fakeClient({ listApprovals: async () => [anApproval()] })} t={t} />);
    expect(await screen.findByText('connector.write')).toBeInTheDocument();
    expect(screen.getByText(`${t('approvalTarget')}: notas/create_note`)).toBeInTheDocument();
  });

  it.each([
    ['actionApprove', 'approve', false],
    ['actionAlwaysAllow', 'approve', true],
    ['actionDeny', 'deny', false],
  ] as const)('%s envia a decisão e recarrega', async (label, decision, remember) => {
    const user = userEvent.setup();
    let list = [anApproval()];
    const client = fakeClient({
      listApprovals: async () => list,
      decideApproval: async () => {
        list = [];
        return anApproval({ status: decision === 'deny' ? 'denied' : 'approved' });
      },
    });
    render(<ApprovalsScreen client={client} t={t} />);
    const item = (await screen.findByText('connector.write')).closest('li') as HTMLElement;
    await user.click(within(item).getByRole('button', { name: t(label) }));
    expect(client.decideApproval).toHaveBeenCalledWith('ap-1', decision, remember);
    expect(await screen.findByText(t('approvalEmpty'))).toBeInTheDocument();
  });

  it('mantém o pedido e mostra o erro quando a decisão falha', async () => {
    const user = userEvent.setup();
    const client = fakeClient({
      listApprovals: async () => [anApproval()],
      decideApproval: async () => Promise.reject(new Error('aprovação expirou')),
    });
    render(<ApprovalsScreen client={client} t={t} />);
    await user.click(await screen.findByRole('button', { name: t('actionApprove') }));
    expect(await screen.findByText('aprovação expirou')).toBeInTheDocument();
    expect(screen.getByText('connector.write')).toBeInTheDocument();
  });

  it('mostra o erro de carregamento', async () => {
    render(
      <ApprovalsScreen
        client={fakeClient({ listApprovals: async () => Promise.reject(new Error('offline')) })}
        t={t}
      />,
    );
    expect(await screen.findByText('offline')).toBeInTheDocument();
  });

  it('decide pelo teclado', async () => {
    const user = userEvent.setup();
    const client = fakeClient({
      listApprovals: async () => [anApproval()],
      decideApproval: () => pending(),
    });
    render(<ApprovalsScreen client={client} t={t} />);
    await screen.findByText('connector.write');
    await user.tab();
    expect(screen.getByRole('button', { name: t('actionApprove') })).toHaveFocus();
    await user.keyboard('{Enter}');
    await waitFor(() => expect(client.decideApproval).toHaveBeenCalledWith('ap-1', 'approve', false));
  });
});
