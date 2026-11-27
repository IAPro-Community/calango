import { useCallback, useEffect, useState, type FormEvent } from 'react';
import type { VaultItem } from '@calango/sdk-ts';
import { bridge } from '../bridge';
import { usePoll } from '../usePoll';
import type { ScreenProps } from './props';
import { Button, Card, Field, Input, PageHeader, Select } from '@';
import { setThemePreference, useThemePreference, type ThemePreference } from '../theme';

export function SettingsScreen({ client, t }: ScreenProps) {
  const load = useCallback(() => client.listVaultItems(), [client]);
  const { data: items, error: loadError, refresh } = usePoll(load, 5_000);
  const maestroLoad = useCallback(() => client.getMaestroStatus(), [client]);
  const { data: maestroStatus, error: maestroLoadError, refresh: refreshMaestro } = usePoll(maestroLoad, 60_000);
  const [maestroBusy, setMaestroBusy] = useState(false);
  const [maestroActionError, setMaestroActionError] = useState<string | null>(null);
  const [name, setName] = useState('');
  const [secret, setSecret] = useState('');
  const [busy, setBusy] = useState(false);
  const [deletingName, setDeletingName] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [launchAtLogin, setLaunchAtLogin] = useState<boolean | null>(null);
  const [launchError, setLaunchError] = useState<string | null>(null);
  const [launchBusy, setLaunchBusy] = useState(false);
  const theme = useThemePreference();

  useEffect(() => {
    let active = true;
    void bridge.getLoginItemSettings().then(
      (settings) => {
        if (active) setLaunchAtLogin(settings.openAtLogin);
      },
      () => {
        if (active) setLaunchError(t('settingsStartupError'));
      },
    );
    return () => {
      active = false;
    };
  }, [t]);

  async function runMaestroAction(action: () => Promise<unknown>) {
    setMaestroBusy(true);
    setMaestroActionError(null);
    try {
      await action();
      refreshMaestro();
    } catch {
      setMaestroActionError(t('maestroActionError'));
    } finally {
      setMaestroBusy(false);
    }
  }

  async function updateLaunchAtLogin(enabled: boolean) {
    setLaunchBusy(true);
    setLaunchError(null);
    try {
      const settings = await bridge.setLaunchAtLogin(enabled);
      setLaunchAtLogin(settings.openAtLogin);
    } catch {
      setLaunchError(t('settingsStartupError'));
    } finally {
      setLaunchBusy(false);
    }
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy || name.trim() === '' || secret.length === 0) {
      return;
    }
    setBusy(true);
    setActionError(null);
    setSaved(false);
    try {
      await client.setVaultSecret(name.trim(), secret);
      setName('');
      setSecret('');
      setSaved(true);
      await refresh();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function remove(item: VaultItem) {
    if (busy || item.name === 'api-token' || !window.confirm(t('settingsSecretDeleteConfirm'))) {
      return;
    }
    setBusy(true);
    setDeletingName(item.name);
    setActionError(null);
    setSaved(false);
    try {
      await client.deleteVaultSecret(item.name);
      await refresh();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : String(err));
    } finally {
      setDeletingName(null);
      setBusy(false);
    }
  }

  return (
    <section className="flex flex-col gap-md">
      <PageHeader title={t('navSettings')} />
      <div className="grid grid-cols-[repeat(auto-fit,minmax(320px,1fr))] items-start gap-md">
        <Card title={t('settingsSecretAddTitle')}>
          <p className="muted">{t('settingsVaultDescription')}</p>
          <form className="flex flex-col gap-sm" onSubmit={(event) => void save(event)}>
            <Field label={t('settingsSecretName')}>
              <Input
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
                pattern="[a-z0-9][a-z0-9_.-]{0,62}"
                maxLength={63}
                disabled={busy}
              />
            </Field>
            <Field label={t('settingsSecretValue')}>
              <Input
                type="password"
                value={secret}
                onChange={(event) => setSecret(event.target.value)}
                required
                autoComplete="off"
                disabled={busy}
              />
            </Field>
            {actionError && (
              <p className="error" role="alert">
                {actionError}
              </p>
            )}
            {saved && <output className="muted">{t('settingsSecretSaved')}</output>}
            <Button type="submit" disabled={busy || name.trim() === '' || secret.length === 0}>
              {busy ? t('commonLoading') : t('commonSave')}
            </Button>
          </form>
        </Card>
        <Card title={t('settingsVault')}>
          {loadError && (
            <p className="error" role="alert">
              {t('settingsVaultLoadError')}
            </p>
          )}
          {items === null ? (
            loadError ? null : (
              <p className="muted">{t('commonLoading')}</p>
            )
          ) : items.length === 0 ? (
            <p className="muted">{t('settingsVaultEmpty')}</p>
          ) : (
            <ul className="list">
              {items.map((item) => (
                <li key={item.name}>
                  <span>
                    <code>{item.name}</code>
                    <span className="muted"> · {new Date(item.updated_at).toLocaleString()}</span>
                  </span>
                  {item.name === 'api-token' ? (
                    <span className="muted">{t('settingsSecretProtected')}</span>
                  ) : (
                    <Button variant="danger" onClick={() => void remove(item)} disabled={busy}>
                      {deletingName === item.name ? t('commonLoading') : t('commonDelete')}
                    </Button>
                  )}
                </li>
              ))}
            </ul>
          )}
        </Card>
        <Card title={t('settingsAppearanceTitle')}>
          <Field label={t('settingsTheme')} hint={t('settingsThemeHint')}>
            <Select value={theme} onChange={(event) => setThemePreference(event.target.value as ThemePreference)}>
              <option value="dark">{t('settingsThemeDark')}</option>
              <option value="light">{t('settingsThemeLight')}</option>
              <option value="system">{t('settingsThemeSystem')}</option>
            </Select>
          </Field>
        </Card>
        <Card title={t('settingsStartupTitle')}>
          <p className="muted">{t('settingsStartupDescription')}</p>
          <label className="check-label">
            <input
              type="checkbox"
              checked={launchAtLogin ?? false}
              onChange={(event) => void updateLaunchAtLogin(event.target.checked)}
              disabled={launchBusy || launchAtLogin === null}
            />
            {t('settingsStartupEnable')}
          </label>
          {launchError && (
            <p className="error" role="alert">
              {launchError}
            </p>
          )}
        </Card>
        <Card title={t('maestroTitle')}>
          <p className="muted">{t('maestroDescription')}</p>
          <p className="muted">{t('maestroLoginNotice')}</p>
          {maestroLoadError && (
            <p className="error" role="alert">
              {t('maestroLoadError')}
            </p>
          )}
          {maestroActionError && (
            <p className="error" role="alert">
              {maestroActionError}
            </p>
          )}
          {maestroStatus === null ? (
            <p className="muted">{t('commonLoading')}</p>
          ) : (
            <>
              <p>
                {t('maestroVersion')}: <code>{maestroStatus.version ?? t('maestroNotInstalled')}</code>
              </p>
              {maestroStatus.update_available && (
                <output className="muted">
                  {t('maestroUpdateAvailable')}: <code>{maestroStatus.update_available.version}</code>
                </output>
              )}
              <label className="check-label">
                <input
                  type="checkbox"
                  checked={maestroStatus.auto_update}
                  onChange={(event) => void runMaestroAction(() => client.setMaestroAutoUpdate(event.target.checked))}
                  disabled={maestroBusy}
                />
                {t('maestroAutoUpdate')}
              </label>
              <div className="actions">
                <Button
                  variant="ghost"
                  onClick={() =>
                    void runMaestroAction(() =>
                      maestroStatus.update_available
                        ? client.installMaestro(maestroStatus.update_available.version)
                        : client.updateMaestro(),
                    )
                  }
                  disabled={maestroBusy}
                >
                  {maestroBusy ? t('commonLoading') : t(maestroStatus.installed ? 'maestroUpdate' : 'maestroInstall')}
                </Button>
                {maestroStatus.rollback_available && (
                  <Button
                    variant="ghost"
                    onClick={() => void runMaestroAction(() => client.rollbackMaestro())}
                    disabled={maestroBusy}
                  >
                    {t('maestroRollback')}
                  </Button>
                )}
              </div>
            </>
          )}
          <p className="muted">
            <a href="https://github.com/IAPro-Community/Orquestrador-Maestro" target="_blank" rel="noreferrer">
              {t('maestroAbout')}
            </a>
          </p>
        </Card>
      </div>
    </section>
  );
}
