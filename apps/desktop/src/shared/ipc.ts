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
  ptyAttach: 'calango:pty-attach',
  ptyInput: 'calango:pty-input',
  ptyResize: 'calango:pty-resize',
  ptyDetach: 'calango:pty-detach',
  ptyOutput: 'calango:pty-output',
  ptyClosed: 'calango:pty-closed',
} as const;

/** Why a terminal socket closed: the session ended on the daemon, or the connection dropped. */
export interface PtyClosed {
  sessionId: string;
  ended: boolean;
  reason?: string;
  exitCode?: number;
}

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
  /**
   * Terminal sessions. The main process opens the WebSocket /v1/pty/{id} with the bearer
   * header (ADR 0015), so the socket never runs in the renderer.
   */
  ptyAttach(sessionId: string): Promise<void>;
  ptyInput(sessionId: string, data: string): void;
  ptyResize(sessionId: string, cols: number, rows: number): void;
  ptyDetach(sessionId: string): void;
  onPtyOutput(listener: (sessionId: string, data: Uint8Array) => void): () => void;
  onPtyClosed(listener: (closed: PtyClosed) => void): () => void;
}
