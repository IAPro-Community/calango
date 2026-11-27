import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent } from 'react';
import { type BrowserTab, type DevServer, PREVIEW_OWNER } from '../../shared/ipc';
import { bridge } from '../bridge';
import { usePoll } from '../usePoll';
import type { ScreenProps } from './props';

// Internal browser panel (F5-T01). The page itself is a native WebContentsView placed by the
// main process over the viewport element below; this screen only sends its bounds and
// drives tabs through the preload bridge. Page content never reaches the renderer.
// The "Project preview" owner (F4-T06) shows preview tabs: local dev servers on loopback,
// in their own profile, confined to the chosen origin by the main process.
export function BrowserScreen({ client, t }: ScreenProps) {
  const browser = bridge.browser;
  const load = useCallback(() => client.listAgents(), [client]);
  const { data: agents, error: agentsError } = usePoll(load, 10_000);
  const [agentId, setAgentId] = useState<string | null>(null);
  const [tabs, setTabs] = useState<BrowserTab[]>([]);
  const [address, setAddress] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [confirmingClear, setConfirmingClear] = useState(false);
  const [servers, setServers] = useState<DevServer[] | null>(null);
  const [detecting, setDetecting] = useState(false);
  const [port, setPort] = useState('');
  const [autoReload, setAutoReload] = useState(false);
  const [blockedUrl, setBlockedUrl] = useState<string | null>(null);
  const isPreview = agentId === PREVIEW_OWNER;
  const viewportRef = useRef<HTMLDivElement>(null);
  const agentRef = useRef<string | null>(null);
  agentRef.current = agentId;

  useEffect(() => {
    // Without agents (or without the daemon) the preview still works: it needs neither.
    if (agentId === null && (agents !== null || agentsError)) {
      setAgentId(agents?.[0]?.id ?? PREVIEW_OWNER);
    }
  }, [agents, agentsError, agentId]);

  useEffect(
    () =>
      browser.onTabs((event) => {
        if (event.agentId === agentRef.current) setTabs(event.tabs);
      }),
    [browser],
  );

  useEffect(
    () =>
      browser.onPreviewBlocked((event) => {
        if (agentRef.current === PREVIEW_OWNER) setBlockedUrl(event.url);
      }),
    [browser],
  );

  useEffect(() => {
    setConfirmingClear(false);
    setError(null);
    setNotice(null);
    setBlockedUrl(null);
    if (agentId === null) {
      setTabs([]);
      return;
    }
    void browser
      .focusAgent(agentId)
      .then(setTabs)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)));
  }, [agentId, browser]);

  const active = tabs.find((tab) => tab.active) ?? null;
  const activeId = active?.id;
  const activeUrl = active?.url;
  // biome-ignore lint/correctness/useExhaustiveDependencies: o endereço acompanha a aba ativa e a URL dela.
  useEffect(() => {
    setAddress(activeUrl ?? '');
  }, [activeId, activeUrl]);

  // Keep the native view over the viewport element; hide it when this screen unmounts.
  // Messages above the viewport move it, so they re-run the sync too.
  const layoutKey = [
    agentId,
    activeId,
    error,
    notice,
    confirmingClear,
    active?.loading,
    blockedUrl,
    detecting,
    servers?.length,
  ].join('|');
  // biome-ignore lint/correctness/useExhaustiveDependencies: layoutKey muda a posição do viewport sem mudar o tamanho.
  useLayoutEffect(() => {
    const element = viewportRef.current;
    if (element === null) {
      void browser.setBounds(null);
      return;
    }
    const sync = () => {
      const rect = element.getBoundingClientRect();
      void browser.setBounds({ x: rect.left, y: rect.top, width: rect.width, height: rect.height });
    };
    sync();
    const observer = new ResizeObserver(sync);
    observer.observe(element);
    window.addEventListener('resize', sync);
    window.addEventListener('scroll', sync, true);
    return () => {
      observer.disconnect();
      window.removeEventListener('resize', sync);
      window.removeEventListener('scroll', sync, true);
      void browser.setBounds(null);
    };
  }, [browser, layoutKey]);

  async function run(action: () => Promise<unknown>, failure?: string): Promise<boolean> {
    setError(null);
    setNotice(null);
    setBlockedUrl(null);
    try {
      await action();
      return true;
    } catch (err) {
      setError(failure ?? (err instanceof Error ? err.message : String(err)));
      return false;
    }
  }

  function submitAddress(event: FormEvent) {
    event.preventDefault();
    if (agentId === null || address.trim() === '') return;
    void run(
      () => (active === null ? browser.openTab(agentId, address) : browser.navigate(active.id, address)),
      t(isPreview ? 'browserPreviewOutside' : 'browserInvalidAddress'),
    );
  }

  async function clearProfile() {
    if (agentId === null) return;
    setConfirmingClear(false);
    if (await run(() => browser.clearProfile(agentId))) {
      setNotice(t(isPreview ? 'browserPreviewCleared' : 'browserProfileCleared'));
    }
  }

  async function detectServers() {
    setDetecting(true);
    await run(async () => setServers(await browser.detectServers()));
    setDetecting(false);
  }

  function openPreview(target: string | number) {
    void run(() => browser.openPreview(target, { autoReload }), t('browserInvalidPort'));
  }

  function submitPort(event: FormEvent) {
    event.preventDefault();
    if (port.trim() !== '') openPreview(port.trim());
  }

  function openOutside(url: string) {
    setBlockedUrl(null);
    void run(() => browser.openExternal(url));
  }

  const agentList = agents ?? [];
  return (
    <section className="screen browser-screen">
      <header className="screen-header">
        <h1>{t('navBrowser')}</h1>
        <label className="browser-agent">
          <span>{t('browserAgentLabel')}</span>
          <select value={agentId ?? ''} onChange={(event) => setAgentId(event.target.value)}>
            {agentList.map((agent) => (
              <option key={agent.id} value={agent.id}>
                {agent.name}
              </option>
            ))}
            <option value={PREVIEW_OWNER}>{t('browserPreviewOption')}</option>
          </select>
        </label>
      </header>
      {agentsError && (
        <p className="error" role="alert">
          {agentsError}
        </p>
      )}
      {agents !== null && agentList.length === 0 && <p className="muted">{t('browserNoAgents')}</p>}
      {agentId !== null && (
        <>
          <p className="muted">{t(isPreview ? 'browserPreviewNote' : 'browserIsolationNote')}</p>
          {isPreview && (
            <div className="browser-preview">
              <div className="browser-preview-row">
                <button type="button" disabled={detecting} onClick={() => void detectServers()}>
                  {t('browserDetectServers')}
                </button>
                <form className="browser-preview-port" onSubmit={submitPort}>
                  <input
                    type="text"
                    inputMode="numeric"
                    aria-label={t('browserPortLabel')}
                    placeholder={t('browserPortLabel')}
                    value={port}
                    spellCheck={false}
                    autoComplete="off"
                    onChange={(event) => setPort(event.target.value)}
                  />
                  <button type="submit">{t('browserOpenPreview')}</button>
                </form>
                <label className="browser-preview-reload">
                  <input
                    type="checkbox"
                    checked={autoReload}
                    onChange={(event) => setAutoReload(event.target.checked)}
                  />
                  <span>{t('browserAutoReload')}</span>
                </label>
              </div>
              {detecting && <output className="muted">{t('browserDetecting')}</output>}
              {!detecting && servers !== null && servers.length === 0 && (
                <p className="muted">{t('browserNoServers')}</p>
              )}
              {!detecting && servers !== null && servers.length > 0 && (
                <ul className="browser-servers">
                  {servers.map((server) => (
                    <li key={server.origin}>
                      <span className="browser-server-port">{server.port}</span>
                      <span className="browser-server-title">{server.title || t('browserServerUntitled')}</span>
                      <span className="muted">
                        {server.kind} · HTTP {server.status}
                      </span>
                      <button type="button" onClick={() => openPreview(server.origin)}>
                        {t('browserOpenPreview')}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
          <div className="browser-tabs">
            <div role="tablist" aria-label={t('browserTabs')} className="browser-tablist">
              {tabs.map((tab) => {
                const label = tab.title || tab.url || t('browserNewTab');
                return (
                  <div key={tab.id} className={tab.active ? 'browser-tab active' : 'browser-tab'}>
                    <button
                      type="button"
                      role="tab"
                      aria-selected={tab.active}
                      title={tab.url}
                      onClick={() => void run(() => browser.activate(tab.id))}
                    >
                      {label}
                    </button>
                    <button
                      type="button"
                      className="browser-icon"
                      aria-label={`${t('browserCloseTab')}: ${label}`}
                      title={t('browserCloseTab')}
                      onClick={() => void run(() => browser.closeTab(tab.id))}
                    >
                      ×
                    </button>
                  </div>
                );
              })}
            </div>
            {!isPreview && (
              <button type="button" onClick={() => void run(() => browser.openTab(agentId))}>
                {t('browserNewTab')}
              </button>
            )}
          </div>
          <form className="browser-toolbar" onSubmit={submitAddress}>
            <button
              type="button"
              className="browser-icon"
              aria-label={t('browserBack')}
              title={t('browserBack')}
              disabled={active === null || !active.canGoBack}
              onClick={() => active !== null && void run(() => browser.back(active.id))}
            >
              ←
            </button>
            <button
              type="button"
              className="browser-icon"
              aria-label={t('browserForward')}
              title={t('browserForward')}
              disabled={active === null || !active.canGoForward}
              onClick={() => active !== null && void run(() => browser.forward(active.id))}
            >
              →
            </button>
            <button
              type="button"
              className="browser-icon"
              aria-label={t('browserReload')}
              title={t('browserReload')}
              disabled={active === null}
              onClick={() => active !== null && void run(() => browser.reload(active.id))}
            >
              ↻
            </button>
            <input
              type="text"
              inputMode="url"
              aria-label={t('browserAddress')}
              placeholder={isPreview ? 'http://localhost:' : 'https://'}
              value={address}
              spellCheck={false}
              autoComplete="off"
              onChange={(event) => setAddress(event.target.value)}
            />
            <button type="submit">{t('browserGo')}</button>
            {confirmingClear ? (
              <>
                <button type="button" className="danger" onClick={() => void clearProfile()}>
                  {t('browserClearProfile')}
                </button>
                <button type="button" onClick={() => setConfirmingClear(false)}>
                  {t('commonCancel')}
                </button>
              </>
            ) : (
              <button type="button" className="danger" onClick={() => setConfirmingClear(true)}>
                {t('browserClearProfile')}
              </button>
            )}
          </form>
          {confirmingClear && (
            <p role="alert">{t(isPreview ? 'browserClearPreviewConfirm' : 'browserClearProfileConfirm')}</p>
          )}
          {blockedUrl && (
            <div className="browser-blocked" role="alert">
              <span>{t('browserPreviewBlocked')}</span>
              <code>{blockedUrl}</code>
              <button type="button" onClick={() => openOutside(blockedUrl)}>
                {t('browserOpenInSystem')}
              </button>
              <button type="button" onClick={() => setBlockedUrl(null)}>
                {t('commonCancel')}
              </button>
            </div>
          )}
          {error && (
            <p className="error" role="alert">
              {error}
            </p>
          )}
          {notice && <output className="muted">{notice}</output>}
          {active?.loading && <output className="muted">{t('browserLoading')}</output>}
          <div className="browser-viewport" ref={viewportRef}>
            {active === null && <p className="muted">{t(isPreview ? 'browserPreviewEmpty' : 'browserEmpty')}</p>}
          </div>
        </>
      )}
    </section>
  );
}
