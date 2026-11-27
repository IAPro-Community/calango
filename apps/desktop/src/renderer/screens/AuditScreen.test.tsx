import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { fakeClient, tr } from '../test/fakes';
import { AuditScreen } from './AuditScreen';

const t = tr('en');

describe('AuditScreen', () => {
  it('começa sem verificação', () => {
    render(<AuditScreen client={fakeClient()} t={t} />);
    expect(screen.getByText(t('auditNeverChecked'))).toBeInTheDocument();
  });

  it('mostra a cadeia válida', async () => {
    const user = userEvent.setup();
    const client = fakeClient({ verifyAudit: async () => ({ ok: true, entries_checked: 12, first_break: null }) });
    render(<AuditScreen client={client} t={t} />);
    await user.click(screen.getByRole('button', { name: t('auditVerify') }));
    expect(await screen.findByRole('heading', { name: t('auditValid') })).toBeInTheDocument();
    expect(screen.getByText(`${t('auditEntriesChecked')}: 12`)).toBeInTheDocument();
  });

  it('mostra onde a cadeia quebrou', async () => {
    const user = userEvent.setup();
    const client = fakeClient({
      verifyAudit: async () => ({
        ok: false,
        entries_checked: 4,
        first_break: { seq: 3, id: 'ev-3', reason: 'hash mismatch' },
      }),
    });
    render(<AuditScreen client={client} t={t} />);
    await user.click(screen.getByRole('button', { name: t('auditVerify') }));
    expect(await screen.findByRole('heading', { name: t('auditInvalid') })).toBeInTheDocument();
    expect(screen.getByText('ev-3')).toBeInTheDocument();
    expect(screen.getByText(`${t('auditBreakReason')}: hash mismatch`)).toBeInTheDocument();
  });

  it('anuncia a falha da verificação', async () => {
    const user = userEvent.setup();
    render(<AuditScreen client={fakeClient({ verifyAudit: async () => Promise.reject(new Error('x')) })} t={t} />);
    await user.click(screen.getByRole('button', { name: t('auditVerify') }));
    expect(await screen.findByRole('alert')).toHaveTextContent(t('auditError'));
  });

  it('exporta e avisa quando a exportação veio vazia', async () => {
    const user = userEvent.setup();
    const createObjectURL = vi.fn(() => 'blob:fake');
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL, revokeObjectURL: vi.fn() }));
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
    const client = fakeClient({ exportAudit: async () => '' });
    render(<AuditScreen client={client} t={t} />);
    await user.click(screen.getByRole('button', { name: t('auditExportCsv') }));
    expect(client.exportAudit).toHaveBeenCalledWith('csv');
    expect(click).toHaveBeenCalled();
    expect(await screen.findByText(t('auditExportEmpty'))).toBeInTheDocument();
    vi.unstubAllGlobals();
  });

  it('anuncia a falha da exportação', async () => {
    const user = userEvent.setup();
    render(<AuditScreen client={fakeClient({ exportAudit: async () => Promise.reject(new Error('x')) })} t={t} />);
    await user.click(screen.getByRole('button', { name: t('auditExportJsonl') }));
    expect(await screen.findByRole('alert')).toHaveTextContent(t('auditExportError'));
  });
});
