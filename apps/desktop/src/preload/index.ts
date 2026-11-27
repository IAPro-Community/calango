import { contextBridge, ipcRenderer } from 'electron';
import type { CalangoBridge, DaemonConnection, DaemonState } from '../shared/ipc';

// Sandboxed preloads cannot require local modules, so the channel names repeat
// the ones in ../shared/ipc.ts.
const CONNECTION = 'calango:connection';
const STATE = 'calango:state';
const STATE_CHANGED = 'calango:state-changed';
const GET_LOGIN_ITEM_SETTINGS = 'calango:get-login-item-settings';
const SET_LAUNCH_AT_LOGIN = 'calango:set-launch-at-login';
const QUICK_CONVERSATION = 'calango:quick-conversation';
const SHORTCUT_UNAVAILABLE = 'calango:shortcut-unavailable';
const shortcutListeners = new Set<(message: string) => void>();
let shortcutPending: string | null = null;

ipcRenderer.on(SHORTCUT_UNAVAILABLE, (_event: unknown, message: string) => {
  if (shortcutListeners.size === 0) {
    shortcutPending = message;
    return;
  }
  for (const listener of shortcutListeners) {
    listener(message);
  }
});
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

const bridge: CalangoBridge = {
  getConnection: () => ipcRenderer.invoke(CONNECTION) as Promise<DaemonConnection>,
  getState: () => ipcRenderer.invoke(STATE) as Promise<DaemonState>,
  getLoginItemSettings: () => ipcRenderer.invoke(GET_LOGIN_ITEM_SETTINGS),
  setLaunchAtLogin: (enabled) => ipcRenderer.invoke(SET_LAUNCH_AT_LOGIN, enabled),
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
  onShortcutUnavailable: (listener) => {
    shortcutListeners.add(listener);
    if (shortcutPending !== null) {
      const pending = shortcutPending;
      shortcutPending = null;
      listener(pending);
    }
    return () => {
      shortcutListeners.delete(listener);
    };
  },
  onState: (listener) => {
    const handler = (_event: unknown, state: DaemonState) => listener(state);
    ipcRenderer.on(STATE_CHANGED, handler);
    return () => {
      ipcRenderer.removeListener(STATE_CHANGED, handler);
    };
  },
};

contextBridge.exposeInMainWorld('calango', bridge);
