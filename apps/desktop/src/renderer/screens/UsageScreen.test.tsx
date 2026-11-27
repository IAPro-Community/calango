import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { aUsageRow, fakeClient, tr } from '../test/fakes';
import { UsageScreen } from './UsageScreen';

const t = tr('en');

describe('UsageScreen', () => {
  it('mostra vazio quando não há consumo', async () => {
    render(<UsageScreen client={fakeClient({ getUsage: async () => [] })} t={t} />);
    expect(await screen.findByText(t('usageEmpty'))).toBeInTheDocument();
  });

  it('mostra a tabela com cabeçalhos e valores', async () => {
    render(
      <UsageScreen
        client={fakeClient({
          getUsage: async () => [aUsageRow(), aUsageRow({ key: '', tokens_in: 3, tokens_out: 1 })],
        })}
        t={t}
      />,
    );
    const table = await screen.findByRole('table');
    for (const key of ['usageKey', 'usageTokensIn', 'usageTokensOut', 'usageCostKnown'] as const) {
      expect(screen.getByRole('columnheader', { name: t(key) })).toBeInTheDocument();
    }
    const rows = screen.getAllByRole('row');
    expect(rows).toHaveLength(3);
    expect(rows[1]).toHaveTextContent('Ana');
    expect(rows[1]).toHaveTextContent('120');
    expect(rows[2]).toHaveTextContent('—');
    expect(table).toBeVisible();
  });

  it('consulta pelo agrupamento e pelo período escolhidos', async () => {
    const user = userEvent.setup();
    const client = fakeClient({ getUsage: async () => [] });
    render(<UsageScreen client={client} t={t} />);
    await screen.findByText(t('usageEmpty'));
    expect(client.getUsage).toHaveBeenLastCalledWith({ groupBy: 'agent', from: undefined, to: undefined });

    await user.selectOptions(screen.getByLabelText(t('usageGroupBy')), 'model');
    await waitFor(() =>
      expect(client.getUsage).toHaveBeenLastCalledWith(expect.objectContaining({ groupBy: 'model' })),
    );

    await user.type(screen.getByLabelText(t('usageFrom')), '2026-11-01');
    await waitFor(() =>
      expect(client.getUsage).toHaveBeenLastCalledWith(
        expect.objectContaining({ from: new Date('2026-11-01T00:00:00').toISOString() }),
      ),
    );
  });

  it('mostra o erro de carregamento', async () => {
    render(
      <UsageScreen client={fakeClient({ getUsage: async () => Promise.reject(new Error('sem ledger')) })} t={t} />,
    );
    expect(await screen.findByText('sem ledger')).toBeInTheDocument();
  });
});
