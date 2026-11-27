import type { components } from './schema';

export type { components };

type Schemas = components['schemas'];

export type Agent = Schemas['Agent'];
export type AgentInput = Schemas['AgentInput'];
export type Conversation = Schemas['Conversation'];
export type Message = Schemas['Message'];
export type Run = Schemas['Run'];
export type RunDetail = Schemas['RunDetail'];
export type Approval = Schemas['Approval'];
export type ApprovalStatus = NonNullable<Approval['status']>;
export type ProviderAccount = Schemas['ProviderAccount'];
export type ProviderAccountInput = Schemas['ProviderAccountInput'];
export type Routine = Schemas['Routine'];
export type RoutineInput = Schemas['RoutineInput'];
export type RoutinePatch = Schemas['RoutinePatch'];
export type AuditVerification = Schemas['AuditVerification'];
export type UsageRow = Schemas['UsageRow'];
export type VaultItem = Schemas['VaultItem'];
export type MaestroStatus = Schemas['MaestroStatus'];
export type MaestroUpdateResult = Schemas['MaestroUpdateResult'];
export type SkillDiscovery = Schemas['SkillDiscovery'];
export type DiscoveredSkill = Schemas['DiscoveredSkill'];
export type SkippedSkill = Schemas['SkippedSkill'];
export type SkillWarning = Schemas['SkillWarning'];
export type ImportedSkill = Schemas['ImportedSkill'];
export type SkillImportInput = Schemas['SkillImportInput'];
export type SkillImportResult = Schemas['SkillImportResult'];

export interface ClientOptions {
  /** Base URL of the local daemon, for example http://127.0.0.1:7317. */
  baseUrl: string;
  /** Local API token. Held in memory only; the client never stores or logs it. */
  token: string;
  fetch?: typeof fetch;
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export interface UsageQuery {
  from?: string;
  to?: string;
  groupBy?: 'agent' | 'routine' | 'project' | 'node' | 'model';
}

export interface CalangoClient {
  listAgents(): Promise<Agent[]>;
  createAgent(input: AgentInput): Promise<Agent>;
  updateAgent(agentId: string, input: AgentInput): Promise<Agent>;
  pauseAll(): Promise<void>;
  resumeAll(): Promise<void>;

  listProviderAccounts(): Promise<ProviderAccount[]>;
  createProviderAccount(input: ProviderAccountInput): Promise<ProviderAccount>;
  /** Stores a secret in the vault. The value is write-only; the API never returns it. */
  setVaultSecret(name: string, value: string): Promise<void>;
  /** Lists metadata only; secret values are never returned. */
  listVaultItems(): Promise<VaultItem[]>;
  /** Deletes a secret; the daemon protects its API-token item. */
  deleteVaultSecret(name: string): Promise<void>;
  listConversations(): Promise<Conversation[]>;
  createConversation(agentId: string, title?: string): Promise<Conversation>;
  listMessages(conversationId: string): Promise<Message[]>;
  /** idempotencyKey makes a retried send return the same message and run. */
  sendMessage(conversationId: string, body: string, idempotencyKey: string): Promise<{ message: Message; run: Run }>;
  getRun(runId: string): Promise<RunDetail>;
  stopRun(runId: string): Promise<void>;

  listApprovals(status?: ApprovalStatus): Promise<Approval[]>;
  decideApproval(approvalId: string, decision: 'approve' | 'deny', remember?: boolean): Promise<Approval>;

  listRoutines(): Promise<Routine[]>;
  createRoutine(input: RoutineInput): Promise<Routine>;
  updateRoutine(routineId: string, patch: RoutinePatch): Promise<Routine>;
  getUsage(query?: UsageQuery): Promise<UsageRow[]>;

  verifyAudit(): Promise<AuditVerification>;
  exportAudit(format: 'jsonl' | 'csv'): Promise<string>;
  getMaestroStatus(): Promise<MaestroStatus>;
  installMaestro(version: string): Promise<MaestroStatus>;
  updateMaestro(): Promise<MaestroUpdateResult>;
  rollbackMaestro(): Promise<MaestroStatus>;
  setMaestroAutoUpdate(enabled: boolean): Promise<MaestroStatus>;

  /** Read-only scan of the authorized skill directories. */
  discoverSkills(): Promise<SkillDiscovery>;
  listImportedSkills(): Promise<ImportedSkill[]>;
  /** Idempotent: the same version returns result "unchanged". */
  importSkill(input: SkillImportInput): Promise<SkillImportResult>;
  /** Removes the node's copy only; the source directory is untouched. */
  removeImportedSkill(skillId: string): Promise<void>;
}

export function createClient(options: ClientOptions): CalangoClient {
  const fetchImpl = options.fetch ?? fetch;
  const base = options.baseUrl.replace(/\/$/, '');

  async function request<T>(
    method: string,
    path: string,
    body?: unknown,
    headers: Record<string, string> = {},
  ): Promise<T> {
    const response = await fetchImpl(base + path, {
      method,
      headers: {
        Authorization: `Bearer ${options.token}`,
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        ...headers,
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    if (!response.ok) {
      let code = 'internal';
      let message = response.statusText || `HTTP ${response.status}`;
      try {
        const parsed = (await response.json()) as Schemas['Error'];
        code = parsed.error.code;
        message = parsed.error.message;
      } catch {
        // A non-JSON error body keeps the status text.
      }
      throw new ApiError(response.status, code, message);
    }
    if (response.status === 204) {
      return undefined as T;
    }
    return (await response.json()) as T;
  }

  async function requestText(path: string): Promise<string> {
    const response = await fetchImpl(base + path, {
      method: 'GET',
      headers: { Authorization: `Bearer ${options.token}` },
    });
    if (!response.ok) {
      let code = 'internal';
      let message = response.statusText || `HTTP ${response.status}`;
      try {
        const parsed = (await response.json()) as Schemas['Error'];
        code = parsed.error.code;
        message = parsed.error.message;
      } catch {
        // A non-JSON error body keeps the status text.
      }
      throw new ApiError(response.status, code, message);
    }
    return response.text();
  }

  return {
    listAgents: () => request<Agent[]>('GET', '/v1/agents'),
    createAgent: (input) => request<Agent>('POST', '/v1/agents', input),
    updateAgent: (agentId, input) => request<Agent>('PUT', `/v1/agents/${encodeURIComponent(agentId)}`, input),
    pauseAll: () => request<void>('POST', '/v1/agents/pause-all'),
    resumeAll: () => request<void>('POST', '/v1/agents/resume-all'),

    listProviderAccounts: () => request<ProviderAccount[]>('GET', '/v1/provider-accounts'),
    createProviderAccount: (input) => request<ProviderAccount>('POST', '/v1/provider-accounts', input),
    setVaultSecret: (name, value) => request<void>('PUT', `/v1/vault/${encodeURIComponent(name)}`, { value }),
    listVaultItems: () => request<VaultItem[]>('GET', '/v1/vault'),
    deleteVaultSecret: (name) => request<void>('DELETE', `/v1/vault/${encodeURIComponent(name)}`),

    listConversations: () => request<Conversation[]>('GET', '/v1/conversations'),
    createConversation: (agentId, title) =>
      request<Conversation>('POST', '/v1/conversations', { agent_id: agentId, title }),
    listMessages: (conversationId) =>
      request<Message[]>('GET', `/v1/conversations/${encodeURIComponent(conversationId)}/messages`),
    sendMessage: (conversationId, body, idempotencyKey) =>
      request<{ message: Message; run: Run }>(
        'POST',
        `/v1/conversations/${encodeURIComponent(conversationId)}/messages`,
        { body },
        { 'Idempotency-Key': idempotencyKey },
      ),
    getRun: (runId) => request<RunDetail>('GET', `/v1/runs/${encodeURIComponent(runId)}`),
    stopRun: (runId) => request<void>('POST', `/v1/runs/${encodeURIComponent(runId)}/stop`),

    listApprovals: (status) => request<Approval[]>('GET', `/v1/approvals${status ? `?status=${status}` : ''}`),
    decideApproval: (approvalId, decision, remember = false) =>
      request<Approval>('POST', `/v1/approvals/${encodeURIComponent(approvalId)}/decide`, { decision, remember }),

    listRoutines: () => request<Routine[]>('GET', '/v1/routines'),
    createRoutine: (input) => request<Routine>('POST', '/v1/routines', input),
    updateRoutine: (routineId, patch) =>
      request<Routine>('PATCH', `/v1/routines/${encodeURIComponent(routineId)}`, patch),

    getUsage: (query = {}) => {
      const params = new URLSearchParams();
      if (query.from) params.set('from', query.from);
      if (query.to) params.set('to', query.to);
      if (query.groupBy) params.set('group_by', query.groupBy);
      const qs = params.toString();
      return request<UsageRow[]>('GET', `/v1/usage${qs ? `?${qs}` : ''}`);
    },
    verifyAudit: () => request<AuditVerification>('GET', '/v1/audit/verify'),
    exportAudit: (format) => requestText(`/v1/audit/export?format=${format}`),
    getMaestroStatus: () => request<MaestroStatus>('GET', '/v1/maestro'),
    installMaestro: (version) => request<MaestroStatus>('POST', '/v1/maestro/install', { version }),
    updateMaestro: () => request<MaestroUpdateResult>('POST', '/v1/maestro/update'),
    rollbackMaestro: () => request<MaestroStatus>('POST', '/v1/maestro/rollback'),
    setMaestroAutoUpdate: (enabled) =>
      request<MaestroStatus>('PATCH', '/v1/maestro/settings', { auto_update: enabled }),

    discoverSkills: () => request<SkillDiscovery>('GET', '/v1/skill-imports/discover'),
    listImportedSkills: () => request<ImportedSkill[]>('GET', '/v1/skill-imports'),
    importSkill: (input) => request<SkillImportResult>('POST', '/v1/skill-imports', input),
    removeImportedSkill: (skillId) => request<void>('DELETE', `/v1/skill-imports/${encodeURIComponent(skillId)}`),
  };
}
