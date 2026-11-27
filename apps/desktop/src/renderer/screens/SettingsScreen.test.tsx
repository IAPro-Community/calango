import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { aMaestroStatus, aVaultItem, fakeBridge, fakeClient, pending, tr } from '../test/fakes';
import { SettingsScreen } from './SettingsScreen';

const t = tr('en');

function setup(overrides: Parameters<typeof fakeClient>[0] = {}) {
  const client = fakeClient({
    listVaultItems: async () => [aVaultItem({ name: 'api-token' }), aVaultItem()],
    getMaestroStatus: async () => aMaestroStatus(),
    ...overrides,
  });
  render(<SettingsScreen client={client} t={t} />);
  return client;
}

function vaultCard() {
  return screen.getByRole('heading', { name: t('settingsVault') }).closest('section') as HTMLElement;
}

describe('SettingsScreen', () => {
  it('mostra carregando enquanto o cofre e o Maestro respondem', () => {
    setup({ listVaultItems: () => pending(), getMaestroStatus: () => pending() });
    expect(screen.getAllByText(t('commonLoading'))).toHaveLength(2);
  });

  it('lista nomes do cofre e protege o token da API', async () => {
    setup();
    expect(await screen.findByText('anthropic-api-key')).toBeInTheDocument();
    const tokenRow = screen.getByText('api-token').closest('li') as HTMLElement;
    expect(within(tokenRow).getByText(t('settingsSecretProtected'))).toBeInTheDocument();
    expect(within(tokenRow).queryByRole('button')).not.toBeInTheDocument();
  });

  it('mostra vazio quando o cofre não tem itens', async () => {
    setup({ listVaultItems: async () => [] });
    expect(await screen.findByText(t('settingsVaultEmpty'))).toBeInTheDocument();
  });

  it('anuncia a falha ao carregar o cofre', async () => {
    setup({ listVaultItems: async () => Promise.reject(new Error('locked')) });
    expect(await within(vaultCard()).findByRole('alert')).toHaveTextContent(t('settingsVaultLoadError'));
  });

  it('salva um segredo sem mostrar o valor e limpa o formulário', async () => {
    const user = userEvent.setup();
    const client = setup({ setVaultSecret: async () => {} });
    await screen.findByText('anthropic-api-key');
    const save = screen.getByRole('button', { name: t('commonSave') });
    expect(save).toBeDisabled();
    await user.type(screen.getByLabelText(t('settingsSecretName')), ' github-token ');
    const value = screen.getByLabelText(t('settingsSecretValue'));
    expect(value).toHaveAttribute('type', 'password');
    await user.type(value, 'ghp_fake');
    await user.keyboard('{Enter}');

    expect(client.setVaultSecret).toHaveBeenCalledWith('github-token', 'ghp_fake');
    expect(await screen.findByText(t('settingsSecretSaved'))).toBeInTheDocument();
    expect(value).toHaveValue('');
    expect(screen.queryByText('ghp_fake')).not.toBeInTheDocument();
  });

  it('anuncia o erro ao salvar', async () => {
    const user = userEvent.setup();
    setup({ setVaultSecret: async () => Promise.reject(new Error('invalid name')) });
    await screen.findByText('anthropic-api-key');
    await user.type(screen.getByLabelText(t('settingsSecretName')), 'x');
    await user.type(screen.getByLabelText(t('settingsSecretValue')), 'y');
    await user.click(screen.getByRole('button', { name: t('commonSave') }));
    expect(await screen.findByRole('alert')).toHaveTextContent('invalid name');
  });

  it('só apaga depois de confirmar', async () => {
    const user = userEvent.setup();
    const client = setup({ deleteVaultSecret: async () => {} });
    const row = (await screen.findByText('anthropic-api-key')).closest('li') as HTMLElement;
    const confirm = vi.spyOn(window, 'confirm').mockReturnValueOnce(false).mockReturnValueOnce(true);

    await user.click(within(row).getByRole('button', { name: t('commonDelete') }));
    expect(confirm).toHaveBeenCalledWith(t('settingsSecretDeleteConfirm'));
    expect(client.deleteVaultSecret).not.toHaveBeenCalled();

    await user.click(within(row).getByRole('button', { name: t('commonDelete') }));
    expect(client.deleteVaultSecret).toHaveBeenCalledWith('anthropic-api-key');
  });

  it('liga a abertura com o sistema pela ponte', async () => {
    const user = userEvent.setup();
    setup();
    const box = screen.getByRole('checkbox', { name: t('settingsStartupEnable') });
    await waitFor(() => expect(box).toBeEnabled());
    expect(box).not.toBeChecked();
    await user.click(box);
    expect(fakeBridge.api.setLaunchAtLogin).toHaveBeenCalledWith(true);
    await waitFor(() => expect(box).toBeChecked());
  });

  it('mostra a versão do Maestro e oferece a atualização disponível', async () => {
    const user = userEvent.setup();
    const client = setup({
      getMaestroStatus: async () =>
        aMaestroStatus({ update_available: { version: '1.3.0' }, rollback_available: true }),
      installMaestro: async () => aMaestroStatus({ version: '1.3.0' }),
    });
    expect(await screen.findByText('1.2.0')).toBeInTheDocument();
    expect(screen.getByText('1.3.0')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: t('maestroRollback') })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: t('maestroUpdate') }));
    expect(client.installMaestro).toHaveBeenCalledWith('1.3.0');
  });

  it('anuncia a falha de uma ação do Maestro', async () => {
    const user = userEvent.setup();
    setup({ updateMaestro: async () => Promise.reject(new Error('npm missing')) });
    await user.click(await screen.findByRole('button', { name: t('maestroUpdate') }));
    expect(await screen.findByRole('alert')).toHaveTextContent(t('maestroActionError'));
  });
});
