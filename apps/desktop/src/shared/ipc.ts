// Shared by the Electron main process and the renderer. The sandboxed preload script
// repeats the channel names and imports only these types, because a sandboxed preload
// cannot require local modules.

export const channels = {
  connection: 'calango:connection',
  state: 'calango:state',
  stateChanged: 'calango:state-changed',
  getLoginItemSettings: 'calango:get-login-item-settings',
  setLaunchAtLogin: 'calango:set-launch-at-login',
  quickConversation: 'calango:quick-conversation',
  nodeInfo: 'calango:node-info',
} as const;

export interface LoginItemSettings {
  openAtLogin: boolean;
}

// NodeInfo is what the pairing screen may show about this computer. It holds no secret.
export interface NodeInfo {
  /** Display label for the pairing QR `node` field; absent when the host name is unusable. */
  label?: string;
}

export interface DaemonConnection {
  baseUrl: string;
  token: string;
}

export type DaemonStatus = 'starting' | 'running' | 'restarting' | 'stopped';

export interface DaemonState {
  status: DaemonStatus;
  detail?: string;
}

export interface CalangoBridge {
  getConnection(): Promise<DaemonConnection>;
  getState(): Promise<DaemonState>;
  getLoginItemSettings(): Promise<LoginItemSettings>;
  setLaunchAtLogin(enabled: boolean): Promise<LoginItemSettings>;
  getNodeInfo(): Promise<NodeInfo>;
  onQuickConversation(listener: () => void): () => void;
  onState(listener: (state: DaemonState) => void): () => void;
}
