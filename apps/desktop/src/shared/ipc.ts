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
  browserList: 'calango:browser-list',
  browserFocusAgent: 'calango:browser-focus-agent',
  browserOpenTab: 'calango:browser-open-tab',
  browserCloseTab: 'calango:browser-close-tab',
  browserActivate: 'calango:browser-activate',
  browserNavigate: 'calango:browser-navigate',
  browserBack: 'calango:browser-back',
  browserForward: 'calango:browser-forward',
  browserReload: 'calango:browser-reload',
  browserSetBounds: 'calango:browser-set-bounds',
  browserClearProfile: 'calango:browser-clear-profile',
  browserTabsChanged: 'calango:browser-tabs-changed',
  browserDetectServers: 'calango:browser-detect-servers',
  browserOpenPreview: 'calango:browser-open-preview',
  browserOpenExternal: 'calango:browser-open-external',
  browserPreviewBlocked: 'calango:browser-preview-blocked',
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

/** A tab of the internal browser, as the renderer sees it (no page content). */
export interface BrowserTab {
  id: string;
  agentId: string;
  url: string;
  title: string;
  loading: boolean;
  canGoBack: boolean;
  canGoForward: boolean;
  active: boolean;
  /** Set on project preview tabs (F4-T06): the loopback origin the tab is confined to. */
  preview?: { origin: string; autoReload: boolean };
}

/** Owner id of preview tabs in the browser bridge (never a valid agent id). */
export const PREVIEW_OWNER = '@preview';
/** A local dev server answering HTTP on a loopback port. */
export interface DevServer {
  port: number;
  /** http://127.0.0.1:PORT or http://[::1]:PORT, whichever answered. */
  origin: string;
  status: number;
  /** Page <title>, as plain text (page data, shown, never interpreted). */
  title: string;
  /** Best-effort guess: vite, next, angular, webpack, express, python or http. */
  kind: string;
}

export interface PreviewOptions {
  autoReload?: boolean;
}

/** A preview tab tried to leave its origin and was kept in place. */
export interface PreviewBlockedEvent {
  tabId: string;
  url: string;
}

export interface BrowserTabsEvent {
  agentId: string;
  tabs: BrowserTab[];
}

/** Rectangle of the browser area in the renderer, in CSS pixels. */
export interface BrowserBounds {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface BrowserBridge {
  listTabs(agentId: string): Promise<BrowserTab[]>;
  focusAgent(agentId: string): Promise<BrowserTab[]>;
  openTab(agentId: string, url?: string): Promise<BrowserTab>;
  closeTab(tabId: string): Promise<void>;
  activate(tabId: string): Promise<BrowserTab>;
  navigate(tabId: string, address: string): Promise<BrowserTab>;
  back(tabId: string): Promise<void>;
  forward(tabId: string): Promise<void>;
  reload(tabId: string): Promise<void>;
  /** null hides the browser view (panel not on screen). */
  setBounds(bounds: BrowserBounds | null): Promise<void>;
  clearProfile(agentId: string): Promise<void>;
  onTabs(listener: (event: BrowserTabsEvent) => void): () => void;
  /** Probes the short list of common dev ports on loopback only. */
  detectServers(): Promise<DevServer[]>;
  /** Opens a preview tab confined to a loopback origin: a port, localhost:PORT or URL. */
  openPreview(target: string | number, options?: PreviewOptions): Promise<BrowserTab>;
  /** Opens, in the system browser, a URL a preview tab was blocked from; the click is the confirmation. */
  openExternal(url: string): Promise<boolean>;
  onPreviewBlocked(listener: (event: PreviewBlockedEvent) => void): () => void;
}

export interface CalangoBridge {
  getConnection(): Promise<DaemonConnection>;
  getState(): Promise<DaemonState>;
  getLoginItemSettings(): Promise<LoginItemSettings>;
  setLaunchAtLogin(enabled: boolean): Promise<LoginItemSettings>;
  onQuickConversation(listener: () => void): () => void;
  onState(listener: (state: DaemonState) => void): () => void;
  browser: BrowserBridge;
}
