import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent } from 'react';
import type { BrowserTab } from '../../shared/ipc';
import { bridge } from '../bridge';
import { usePoll } from '../usePoll';
import type { ScreenProps } from './props';

// Internal browser panel (F5-T01). The page itself is a native WebContentsView placed by the
// main process over the viewport element below; this screen only sends its bounds and
// drives tabs through the preload bridge. Page content never reaches the renderer.
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
  const viewportRef = useRef<HTMLDivElement>(null);
  const agentRef = useRef<string | null>(null);
  agentRef.current = agentId;

  useEffect(() => {
    if (agentId === null && agents !== null && agents.length > 0) {
      setAgentId(agents[0]?.id ?? null);
    }
  }, [agents, agentId]);

  useEffect(
    () =>
      browser.onTabs((event) => {
        if (event.agentId === agentRef.current) setTabs(event.tabs);
      }),
    [browser],
  );

  useEffect(() => {
    setConfirmingClear(false);
    setError(null);
    setNotice(null);
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
  const layoutKey = [agentId, activeId, error, notice, confirmingClear, active?.loading].join('|');
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
      t('browserInvalidAddress'),
    );
  }

  async function clearProfile() {
    if (agentId === null) return;
    setConfirmingClear(false);
    if (await run(() => browser.clearProfile(agentId))) {
      setNotice(t('browserProfileCleared'));
    }
  }

  const agentList = agents ?? [];
  return (
    <section className="screen browser-screen">
      <header className="screen-header">
        <h1>{t('navBrowser')}</h1>
        {agentList.length > 0 && (
          <label className="browser-agent">
            <span>{t('browserAgentLabel')}</span>
            <select value={agentId ?? ''} onChange={(event) => setAgentId(event.target.value)}>
              {agentList.map((agent) => (
                <option key={agent.id} value={agent.id}>
                  {agent.name}
                </option>
              ))}
            </select>
          </label>
        )}
      </header>
      {agentsError && <p className="error" role="alert">{agentsError}</p>}
      {agents !== null && agentList.length === 0 && <p className="muted">{t('browserNoAgents')}</p>}
      {agentId !== null && (
        <>
          <p className="muted">{t('browserIsolationNote')}</p>
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
            <button type="button" onClick={() => void run(() => browser.openTab(agentId))}>
              {t('browserNewTab')}
            </button>
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
              placeholder="https://"
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
          {confirmingClear && <p role="alert">{t('browserClearProfileConfirm')}</p>}
          {error && (
            <p className="error" role="alert">
              {error}
            </p>
          )}
          {notice && <output className="muted">{notice}</output>}
          {active?.loading && <output className="muted">{t('browserLoading')}</output>}
          <div className="browser-viewport" ref={viewportRef}>
            {active === null && <p className="muted">{t('browserEmpty')}</p>}
          </div>
        </>
      )}
    </section>
  );
}
