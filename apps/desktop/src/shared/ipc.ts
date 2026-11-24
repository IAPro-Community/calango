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
} as const;

export interface LoginItemSettings {
  openAtLogin: boolean;
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
  onQuickConversation(listener: () => void): () => void;
  onState(listener: (state: DaemonState) => void): () => void;
}
