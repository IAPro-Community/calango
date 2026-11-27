import { contextBridge, ipcRenderer } from 'electron';
import type {
  BrowserBridge,
  BrowserTabsEvent,
  CalangoBridge,
  DaemonConnection,
  DaemonState,
  NodeInfo,
  PreviewBlockedEvent,
} from '../shared/ipc';

// Sandboxed preloads cannot require local modules, so the channel names repeat
// the ones in ../shared/ipc.ts.
const CONNECTION = 'calango:connection';
const STATE = 'calango:state';
const STATE_CHANGED = 'calango:state-changed';
const GET_LOGIN_ITEM_SETTINGS = 'calango:get-login-item-settings';
const SET_LAUNCH_AT_LOGIN = 'calango:set-launch-at-login';
const QUICK_CONVERSATION = 'calango:quick-conversation';
const NODE_INFO = 'calango:node-info';
const BROWSER_TABS_CHANGED = 'calango:browser-tabs-changed';
const BROWSER_PREVIEW_BLOCKED = 'calango:browser-preview-blocked';
const quickConversationListeners = new Set<() => void>();
let quickConversationPending = false;

ipcRenderer.on(QUICK_CONVERSATION, () => {
  if (quickConversationListeners.size === 0) {
    quickConversationPending = true;
    return;
  }
  for (const listener of quickConversationListeners) {
    listener();
  }
});

const browser: BrowserBridge = {
  listTabs: (agentId) => ipcRenderer.invoke('calango:browser-list', agentId),
  focusAgent: (agentId) => ipcRenderer.invoke('calango:browser-focus-agent', agentId),
  openTab: (agentId, url) => ipcRenderer.invoke('calango:browser-open-tab', agentId, url),
  closeTab: (tabId) => ipcRenderer.invoke('calango:browser-close-tab', tabId),
  activate: (tabId) => ipcRenderer.invoke('calango:browser-activate', tabId),
  navigate: (tabId, address) => ipcRenderer.invoke('calango:browser-navigate', tabId, address),
  back: (tabId) => ipcRenderer.invoke('calango:browser-back', tabId),
  forward: (tabId) => ipcRenderer.invoke('calango:browser-forward', tabId),
  reload: (tabId) => ipcRenderer.invoke('calango:browser-reload', tabId),
  setBounds: (bounds) => ipcRenderer.invoke('calango:browser-set-bounds', bounds),
  clearProfile: (agentId) => ipcRenderer.invoke('calango:browser-clear-profile', agentId),
  onTabs: (listener) => {
    const handler = (_event: unknown, payload: BrowserTabsEvent) => listener(payload);
    ipcRenderer.on(BROWSER_TABS_CHANGED, handler);
    return () => {
      ipcRenderer.removeListener(BROWSER_TABS_CHANGED, handler);
    };
  },
  detectServers: () => ipcRenderer.invoke('calango:browser-detect-servers'),
  openPreview: (target, options) => ipcRenderer.invoke('calango:browser-open-preview', target, options),
  openExternal: (url) => ipcRenderer.invoke('calango:browser-open-external', url),
  onPreviewBlocked: (listener) => {
    const handler = (_event: unknown, payload: PreviewBlockedEvent) => listener(payload);
    ipcRenderer.on(BROWSER_PREVIEW_BLOCKED, handler);
    return () => {
      ipcRenderer.removeListener(BROWSER_PREVIEW_BLOCKED, handler);
    };
  },
};

const bridge: CalangoBridge = {
  getConnection: () => ipcRenderer.invoke(CONNECTION) as Promise<DaemonConnection>,
  getState: () => ipcRenderer.invoke(STATE) as Promise<DaemonState>,
  getLoginItemSettings: () => ipcRenderer.invoke(GET_LOGIN_ITEM_SETTINGS),
  setLaunchAtLogin: (enabled) => ipcRenderer.invoke(SET_LAUNCH_AT_LOGIN, enabled),
  getNodeInfo: () => ipcRenderer.invoke(NODE_INFO) as Promise<NodeInfo>,
  onQuickConversation: (listener) => {
    quickConversationListeners.add(listener);
    if (quickConversationPending) {
      quickConversationPending = false;
      listener();
    }
    return () => {
      quickConversationListeners.delete(listener);
    };
  },
  onState: (listener) => {
    const handler = (_event: unknown, state: DaemonState) => listener(state);
    ipcRenderer.on(STATE_CHANGED, handler);
    return () => {
      ipcRenderer.removeListener(STATE_CHANGED, handler);
    };
  },
  browser,
};

contextBridge.exposeInMainWorld('calango', bridge);
