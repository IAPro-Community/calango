import { useEffect, useMemo, useState } from 'react';
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
import { SettingsScreen } from './screens/SettingsScreen';

type Section = 'agents' | 'conversations' | 'approvals' | 'routines' | 'usage' | 'audit' | 'settings';

const SECTIONS: { id: Section; key: MessageKey }[] = [
  { id: 'agents', key: 'navAgents' },
  { id: 'conversations', key: 'navConversations' },
  { id: 'approvals', key: 'navApprovals' },
  { id: 'routines', key: 'navRoutines' },
  { id: 'usage', key: 'navUsage' },
  { id: 'audit', key: 'navAudit' },
  { id: 'settings', key: 'navSettings' },
];

export function App() {
  const [locale] = useState(detectLocale);
  const t = useMemo(() => translator(locale), [locale]);
  const [section, setSection] = useState<Section>('agents');
  useEffect(() => bridge.onQuickConversation(() => setSection('conversations')), []);
  // The accelerator is registered by the OS, so a conflict can only be reported
  // from the main process. The banner keeps the failure visible instead of a log.
  const [notice, setNotice] = useState<string | null>(null);
  useEffect(() => bridge.onShortcutUnavailable((message) => setNotice(message)), []);
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
            onClick={() => setSection(item.id)}
          >
            {t(item.key)}
          </button>
        ))}
      </nav>
      <main className="content">
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
      <section className="status">
        <h1>{t('commonLoading')}</h1>
      </section>
    );
  }
  if (state.status === 'restarting') {
    return (
      <section className="status">
        <h1>{t('daemonRetrying')}</h1>
        {state.detail && <p className="muted">{state.detail}</p>}
      </section>
    );
  }
  return (
    <section className="status">
      <h1>{t('daemonUnavailable')}</h1>
      {state.detail && <p className="muted">{state.detail}</p>}
    </section>
  );
}
