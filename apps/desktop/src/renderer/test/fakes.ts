// Typed fakes for renderer tests: an SDK client whose every method fails until a test
// overrides it, sample records, and a controllable preload bridge.
import type {
  Agent,
  Approval,
  CalangoClient,
  Conversation,
  MaestroStatus,
  Message,
  ProviderAccount,
  Routine,
  RunDetail,
  UsageRow,
  VaultItem,
} from '@calango/sdk-ts';
import { vi } from 'vitest';
import type { CalangoBridge, DaemonConnection, DaemonState, LoginItemSettings } from '../../shared/ipc';
import { translator, type Locale, type Translate } from '../i18n';

type Methods<T> = { [K in keyof T]: T[K] extends (...args: infer A) => infer R ? (...args: A) => R : never };

/**
 * fakeClient returns a CalangoClient whose methods are vi.fn mocks. A method a test
 * does not override rejects, so an unexpected API call fails loudly.
 */
export function fakeClient(overrides: Partial<Methods<CalangoClient>> = {}) {
  const methods: (keyof CalangoClient)[] = [
    'listAgents',
    'createAgent',
    'updateAgent',
    'pauseAll',
    'resumeAll',
    'listProviderKinds',
    'listProviderAccounts',
    'listProviderModels',
    'createProviderAccount',
    'setVaultSecret',
    'listVaultItems',
    'deleteVaultSecret',
    'listConversations',
    'createConversation',
    'listMessages',
    'sendMessage',
    'getRun',
    'stopRun',
    'listApprovals',
    'decideApproval',
    'listRoutines',
    'createRoutine',
    'updateRoutine',
    'getUsage',
    'verifyAudit',
    'exportAudit',
    'getMaestroStatus',
    'installMaestro',
    'updateMaestro',
    'rollbackMaestro',
    'setMaestroAutoUpdate',
  ];
  const client = {} as Record<keyof CalangoClient, ReturnType<typeof vi.fn>>;
  for (const name of methods) {
    client[name] = vi.fn(overrides[name] ?? (() => Promise.reject(new Error(`unexpected call: ${name}`))));
  }
  client.listProviderKinds = vi.fn(
    overrides.listProviderKinds ??
      (async () => [
        { kind: 'codex', display_name: 'Codex', auth_kind: 'oauth_cli' },
        { kind: 'anthropic', display_name: 'Anthropic', auth_kind: 'api_key' },
      ]),
  );
  return client as unknown as { [K in keyof CalangoClient]: ReturnType<typeof vi.fn<CalangoClient[K]>> };
}

export function tr(locale: Locale = 'en'): Translate {
  return translator(locale);
}

const at = '2026-11-25T12:00:00Z';

export function anAgent(patch: Partial<Agent> = {}): Agent {
  return {
    id: 'agent-1',
    name: 'Ana',
    role: 'pesquisa',
    token_limit_month: 1_000_000,
    providers: [{ account_id: 'acc-1', model: 'gpt-fake', account: 'codex/default' }],
    run_on: 'auto',
    approval_level: 'sensitive',
    paused: false,
    tokens_used_month: 0,
    maestro_enabled: false,
    ...patch,
  };
}

export function anAccount(patch: Partial<ProviderAccount> = {}): ProviderAccount {
  return {
    id: 'acc-1',
    kind: 'codex',
    label: 'default',
    key: 'codex/default',
    setup: 'codex login',
    created_at: at,
    updated_at: at,
    ...patch,
  };
}

export function aConversation(patch: Partial<Conversation> = {}): Conversation {
  return { id: 'conv-1', kind: 'dm', title: 'Planejamento', created_at: at, ...patch };
}

export function aMessage(patch: Partial<Message> = {}): Message {
  return {
    id: 'msg-1',
    conversation_id: 'conv-1',
    author_kind: 'person',
    author_id: 'me',
    body: 'Olá',
    created_at: at,
    ...patch,
  };
}

export function aRun(patch: Partial<RunDetail> = {}): RunDetail {
  return {
    id: 'run-1',
    agent_id: 'agent-1',
    conversation_id: 'conv-1',
    source_message_id: 'msg-1',
    trigger: 'message',
    status: 'running',
    tokens_in: 0,
    tokens_out: 0,
    steps: [],
    ...patch,
  };
}

export function anApproval(patch: Partial<Approval> = {}): Approval {
  return {
    id: 'ap-1',
    run_id: 'run-1',
    agent_id: 'agent-1',
    action: 'connector.write',
    target: 'notas/create_note',
    status: 'pending',
    ...patch,
  };
}

export function aRoutine(patch: Partial<Routine> = {}): Routine {
  return {
    id: 'rt-1',
    agent_id: 'agent-1',
    name: 'Resumo diário',
    prompt: 'Resuma o dia',
    trigger_kind: 'interval',
    schedule: '1h',
    run_on: 'auto',
    notify_on: 'news',
    max_retries: 0,
    paused: false,
    next_run_at: at,
    ...patch,
  };
}

export function aUsageRow(patch: Partial<UsageRow> = {}): UsageRow {
  return {
    key: 'Ana',
    tokens_in: 120,
    tokens_out: 45,
    cost_micros: 0,
    cost_known_micros: 0,
    cost_estimated_micros: 0,
    cost_unknown_rows: 1,
    ...patch,
  };
}

export function aVaultItem(patch: Partial<VaultItem> = {}): VaultItem {
  return { name: 'anthropic-api-key', created_at: at, updated_at: at, ...patch };
}

export function aMaestroStatus(patch: Partial<MaestroStatus> = {}): MaestroStatus {
  return { installed: true, version: '1.2.0', auto_update: true, rollback_available: false, ...patch };
}

/** never resolves; keeps a screen in its loading state. */
export function pending<T>(): Promise<T> {
  return new Promise<T>(() => {});
}

// The preload bridge fake. Tests drive daemon state with emitState.
function createFakeBridge() {
  let state: DaemonState = { status: 'starting' };
  let connection: DaemonConnection | null = null;
  const stateListeners = new Set<(state: DaemonState) => void>();
  const quickListeners = new Set<() => void>();
  let login: LoginItemSettings = { openAtLogin: false };

  const api: CalangoBridge = {
    getConnection: vi.fn(async () => {
      if (connection === null) {
        throw new Error('daemon is not running');
      }
      return connection;
    }),
    getState: vi.fn(async () => state),
    getLoginItemSettings: vi.fn(async () => login),
    setLaunchAtLogin: vi.fn(async (enabled: boolean) => {
      login = { openAtLogin: enabled };
      return login;
    }),
    getNodeInfo: vi.fn(async () => ({})),
    onQuickConversation: (listener) => {
      quickListeners.add(listener);
      return () => quickListeners.delete(listener);
    },
    onShortcutUnavailable: () => () => {},
    onState: (listener) => {
      stateListeners.add(listener);
      return () => stateListeners.delete(listener);
    },
    ptyAttach: vi.fn(async () => {}),
    ptyInput: vi.fn(() => {}),
    ptyResize: vi.fn(() => {}),
    ptyDetach: vi.fn(() => {}),
    onPtyOutput: () => () => {},
    onPtyClosed: () => () => {},
    // The browser pane lives in Electron; component tests never mount it.
    browser: {
      listTabs: vi.fn(async () => []),
      focusAgent: vi.fn(async () => []),
      openTab: vi.fn(async (): Promise<never> => {
        throw new Error('browser is not available in component tests');
      }),
      closeTab: vi.fn(async () => {}),
      activate: vi.fn(async (): Promise<never> => {
        throw new Error('browser is not available in component tests');
      }),
      navigate: vi.fn(async (): Promise<never> => {
        throw new Error('browser is not available in component tests');
      }),
      back: vi.fn(async () => {}),
      forward: vi.fn(async () => {}),
      reload: vi.fn(async () => {}),
      setBounds: vi.fn(async () => {}),
      clearProfile: vi.fn(async () => {}),
      onTabs: () => () => {},
      detectServers: vi.fn(async () => []),
      openPreview: vi.fn(async (): Promise<never> => {
        throw new Error('browser is not available in component tests');
      }),
      openExternal: vi.fn(async () => false),
      onPreviewBlocked: () => () => {},
    },
  };

  return {
    api,
    emitState(next: DaemonState, nextConnection: DaemonConnection | null = connection) {
      state = next;
      connection = next.status === 'running' ? nextConnection : null;
      for (const listener of stateListeners) {
        listener(next);
      }
    },
    setConnection(next: DaemonConnection | null) {
      connection = next;
    },
    quickConversation() {
      for (const listener of quickListeners) {
        listener();
      }
    },
    reset() {
      state = { status: 'starting' };
      connection = null;
      login = { openAtLogin: false };
      stateListeners.clear();
      quickListeners.clear();
      vi.mocked(api.getConnection).mockClear();
      vi.mocked(api.getState).mockClear();
    },
  };
}

export const fakeBridge = createFakeBridge();
