import { lazy, type ReactNode, Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createClient, type CalangoClient } from '@calango/sdk-ts';
import { NavItem, NodePill, type NodeState } from '@';
import type { DaemonState } from '../shared/ipc';
import { bridge } from './bridge';
import { detectLocale, translator, type Translate, type MessageKey } from './i18n';
import { useDaemon } from './useDaemon';
import { usePoll } from './usePoll';
import { AgentsScreen } from './screens/AgentsScreen';
import { ApprovalsScreen } from './screens/ApprovalsScreen';
import { ConversationsScreen } from './screens/ConversationsScreen';
import { RoutinesScreen } from './screens/RoutinesScreen';
import { UsageScreen } from './screens/UsageScreen';
import { AuditScreen } from './screens/AuditScreen';
import { BrowserScreen } from './screens/BrowserScreen';
import { SettingsScreen } from './screens/SettingsScreen';
import { DevicesScreen } from './screens/DevicesScreen';
import { SkillsScreen } from './screens/SkillsScreen';

// xterm.js is large; load the terminal screen only when it is opened.
const TerminalScreen = lazy(() => import('./screens/TerminalScreen').then((m) => ({ default: m.TerminalScreen })));

type Section =
  | 'agents'
  | 'conversations'
  | 'approvals'
  | 'routines'
  | 'skills'
  | 'terminal'
  | 'usage'
  | 'audit'
  | 'devices'
  | 'browser'
  | 'settings';

// Menu groups follow the Figma desktop sidebar (D1).
const NAV_GROUPS: { key: MessageKey; items: { id: Section; key: MessageKey }[] }[] = [
  {
    key: 'navGroupMain',
    items: [
      { id: 'conversations', key: 'navConversations' },
      { id: 'approvals', key: 'navApprovals' },
      { id: 'routines', key: 'navRoutines' },
    ],
  },
  {
    key: 'navGroupManage',
    items: [
      { id: 'agents', key: 'navAgents' },
      { id: 'skills', key: 'navSkills' },
      { id: 'usage', key: 'navUsage' },
      { id: 'audit', key: 'navAudit' },
      { id: 'settings', key: 'navSettings' },
    ],
  },
  {
    key: 'navGroupTools',
    items: [
      { id: 'terminal', key: 'navTerminal' },
      { id: 'browser', key: 'navBrowser' },
    ],
  },
];

const NODE_STATE: Record<DaemonState['status'], { state: NodeState; key: MessageKey }> = {
  running: { state: 'online', key: 'nodeStateOnline' },
  starting: { state: 'working', key: 'nodeStateStarting' },
  restarting: { state: 'working', key: 'nodeStateStarting' },
  stopped: { state: 'offline', key: 'nodeStateOffline' },
};

export function App() {
  const [locale] = useState(detectLocale);
  const t = useMemo(() => translator(locale), [locale]);
  const [section, setSection] = useState<Section>('agents');
  const mainRef = useRef<HTMLElement>(null);
  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);
  useEffect(() => bridge.onQuickConversation(() => setSection('conversations')), []);
  // The accelerator is registered by the OS, so a conflict can only be reported
  // from the main process. The banner keeps the failure visible instead of a log.
  const [notice, setNotice] = useState<string | null>(null);
  useEffect(() => bridge.onShortcutUnavailable((message) => setNotice(message)), []);

  // Leva o foco para o conteúdo ao trocar de seção, para quem navega por teclado
  // ou leitor de tela não precisar atravessar o menu de novo.
  function openSection(next: Section) {
    setSection(next);
    window.requestAnimationFrame(() => mainRef.current?.focus());
  }
  const { state, connection } = useDaemon();

  const client = useMemo(
    () => (connection === null ? null : createClient({ baseUrl: connection.baseUrl, token: connection.token })),
    [connection],
  );

  const node = NODE_STATE[state.status];

  // Contador do menu: aprovações esperando decisão.
  const loadPending = useCallback(
    async () => (client === null ? 0 : (await client.listApprovals('pending')).length),
    [client],
  );
  const { data: pendingApprovals } = usePoll(loadPending, 5_000);
  const counts: Partial<Record<Section, number>> = { approvals: pendingApprovals ?? 0 };

  return (
    <div className="grid h-screen grid-cols-[248px_minmax(0,1fr)] bg-bg font-body text-ink">
      <nav
        className="flex flex-col gap-md overflow-y-auto border-r border-line bg-card px-md py-6"
        aria-label={t('appName')}
      >
        <div className="flex items-center gap-xs px-2xs">
          <BrandMark />
          <div className="flex flex-col">
            <span className="font-display text-[20px] leading-6 font-bold text-ink">{t('appName')}</span>
            <span className="font-mono text-[11.5px] leading-[15px] text-ink-soft">{t('brandSubtitle')}</span>
          </div>
        </div>
        {NAV_GROUPS.map((group) => (
          <NavGroup key={group.key} title={t(group.key)}>
            {group.items.map((item) => (
              <li key={item.id}>
                <NavItem
                  active={item.id === section}
                  count={counts[item.id]}
                  countLabel={`${counts[item.id] ?? 0} ${t('navPending')}`}
                  onClick={() => openSection(item.id)}
                >
                  {t(item.key)}
                </NavItem>
              </li>
            ))}
          </NavGroup>
        ))}
        <NavGroup title={t('navGroupNodes')}>
          <li>
            <NodePill kind="pc" state={node.state} label={t('nodeThisComputer')} status={t(node.key)} />
          </li>
          <li>
            <NavItem active={section === 'devices'} onClick={() => openSection('devices')}>
              {t('navDevices')}
            </NavItem>
          </li>
        </NavGroup>
      </nav>
      <main className="min-w-0 overflow-y-auto px-2xl py-6 focus:outline-none" ref={mainRef} tabIndex={-1}>
        {notice !== null ? (
          <output className="notice">
            {notice}
            <button type="button" onClick={() => setNotice(null)} aria-label={t('commonCancel')}>
              ×
            </button>
          </output>
        ) : null}
        {client === null ? (
          <DaemonStatusView state={state} t={t} />
        ) : (
          <SectionView section={section} client={client} t={t} />
        )}
      </main>
    </div>
  );
}

function NavGroup({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2xs">
      <h2 className="m-0 px-2xs font-mono text-[11px] leading-[14px] font-bold tracking-[0.08em] text-ink-soft uppercase">
        {title}
      </h2>
      <ul className="m-0 flex list-none flex-col gap-2xs p-0">{children}</ul>
    </section>
  );
}

// BrandMark is the post-it square of the Figma brand block; decorative, the name is next to it.
function BrandMark() {
  return (
    <svg aria-hidden="true" viewBox="0 0 32 32" className="size-8 shrink-0">
      <rect width="32" height="32" rx="8" fill="var(--calango-color-postit)" />
      <path
        d="M8 22c3-1 4-4 7-6s6-1 8-4M19 9l3 3M12 21l-2 3M17 17l1 3"
        fill="none"
        stroke="var(--calango-color-leaf)"
        strokeWidth="2.4"
        strokeLinecap="round"
      />
    </svg>
  );
}

function SectionView({ section, client, t }: { section: Section; client: CalangoClient; t: Translate }) {
  switch (section) {
    case 'agents':
      return <AgentsScreen client={client} t={t} />;
    case 'conversations':
      return <ConversationsScreen client={client} t={t} />;
    case 'approvals':
      return <ApprovalsScreen client={client} t={t} />;
    case 'routines':
      return <RoutinesScreen client={client} t={t} />;
    case 'terminal':
      return (
        <Suspense fallback={<p className="muted">{t('commonLoading')}</p>}>
          <TerminalScreen client={client} t={t} />
        </Suspense>
      );
    case 'browser':
      return <BrowserScreen client={client} t={t} />;
    case 'skills':
      return <SkillsScreen client={client} t={t} />;
    case 'usage':
      return <UsageScreen client={client} t={t} />;
    case 'audit':
      return <AuditScreen client={client} t={t} />;
    case 'devices':
      return <DevicesScreen client={client} t={t} />;
    case 'settings':
      return <SettingsScreen client={client} t={t} />;
  }
}

function DaemonStatusView({ state, t }: { state: DaemonState; t: Translate }) {
  if (state.status === 'starting') {
    return (
      <section className="status" aria-live="polite">
        <h1>{t('commonLoading')}</h1>
      </section>
    );
  }
  if (state.status === 'restarting') {
    return (
      <section className="status" aria-live="polite">
        <h1>{t('daemonRetrying')}</h1>
        {state.detail && <p className="muted">{state.detail}</p>}
      </section>
    );
  }
  return (
    <section className="status" aria-live="polite">
      <h1>{t('daemonUnavailable')}</h1>
      {state.detail && <p className="muted">{state.detail}</p>}
    </section>
  );
}
