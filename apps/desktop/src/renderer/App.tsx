import { lazy, Suspense, useEffect, useMemo, useRef, useState } from 'react';
import { createClient, type CalangoClient } from '@calango/sdk-ts';
import type { DaemonState } from '../shared/ipc';
import { bridge } from './bridge';
import { detectLocale, translator, type Translate, type MessageKey } from './i18n';
import { useDaemon } from './useDaemon';
import { AgentsScreen } from './screens/AgentsScreen';
import { ApprovalsScreen } from './screens/ApprovalsScreen';
import { ConversationsScreen } from './screens/ConversationsScreen';
import { RoutinesScreen } from './screens/RoutinesScreen';
import { UsageScreen } from './screens/UsageScreen';
import { AuditScreen } from './screens/AuditScreen';
import { BrowserScreen } from './screens/BrowserScreen';
import { SettingsScreen } from './screens/SettingsScreen';
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
  | 'browser'
  | 'usage'
  | 'audit'
  | 'settings';

const SECTIONS: { id: Section; key: MessageKey }[] = [
  { id: 'agents', key: 'navAgents' },
  { id: 'conversations', key: 'navConversations' },
  { id: 'approvals', key: 'navApprovals' },
  { id: 'routines', key: 'navRoutines' },
  { id: 'skills', key: 'navSkills' },
  { id: 'terminal', key: 'navTerminal' },
  { id: 'browser', key: 'navBrowser' },
  { id: 'usage', key: 'navUsage' },
  { id: 'audit', key: 'navAudit' },
  { id: 'settings', key: 'navSettings' },
];

export function App() {
  const [locale] = useState(detectLocale);
  const t = useMemo(() => translator(locale), [locale]);
  const [section, setSection] = useState<Section>('agents');
  const mainRef = useRef<HTMLElement>(null);
  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);
  useEffect(() => bridge.onQuickConversation(() => setSection('conversations')), []);

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

  return (
    <div className="shell">
      <nav className="sidebar" aria-label={t('appName')}>
        <div className="brand">{t('appName')}</div>
        {SECTIONS.map((item) => (
          <button
            key={item.id}
            type="button"
            className={item.id === section ? 'nav-item active' : 'nav-item'}
            aria-current={item.id === section ? 'page' : undefined}
            onClick={() => openSection(item.id)}
          >
            {t(item.key)}
          </button>
        ))}
      </nav>
      <main className="content" ref={mainRef} tabIndex={-1}>
        {client === null ? (
          <DaemonStatusView state={state} t={t} />
        ) : (
          <SectionView section={section} client={client} t={t} />
        )}
      </main>
    </div>
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
    case 'skills':
      return <SkillsScreen client={client} t={t} />;
    case 'terminal':
      return (
        <Suspense fallback={<p className="muted">{t('commonLoading')}</p>}>
          <TerminalScreen client={client} t={t} />
        </Suspense>
      );
    case 'browser':
      return <BrowserScreen client={client} t={t} />;
    case 'usage':
      return <UsageScreen client={client} t={t} />;
    case 'audit':
      return <AuditScreen client={client} t={t} />;
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
