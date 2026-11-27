import { contextBridge, ipcRenderer } from 'electron';
import type { CalangoBridge, DaemonConnection, DaemonState, PtyClosed } from '../shared/ipc';

// Sandboxed preloads cannot require local modules, so the channel names repeat
// the ones in ../shared/ipc.ts.
const CONNECTION = 'calango:connection';
const STATE = 'calango:state';
const STATE_CHANGED = 'calango:state-changed';
const GET_LOGIN_ITEM_SETTINGS = 'calango:get-login-item-settings';
const SET_LAUNCH_AT_LOGIN = 'calango:set-launch-at-login';
const QUICK_CONVERSATION = 'calango:quick-conversation';
const PTY_ATTACH = 'calango:pty-attach';
const PTY_INPUT = 'calango:pty-input';
const PTY_RESIZE = 'calango:pty-resize';
const PTY_DETACH = 'calango:pty-detach';
const PTY_OUTPUT = 'calango:pty-output';
const PTY_CLOSED = 'calango:pty-closed';
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
  onState: (listener) => {
    const handler = (_event: unknown, state: DaemonState) => listener(state);
    ipcRenderer.on(STATE_CHANGED, handler);
    return () => {
      ipcRenderer.removeListener(STATE_CHANGED, handler);
    };
  },
  ptyAttach: (sessionId) => ipcRenderer.invoke(PTY_ATTACH, sessionId) as Promise<void>,
  ptyInput: (sessionId, data) => ipcRenderer.send(PTY_INPUT, sessionId, data),
  ptyResize: (sessionId, cols, rows) => ipcRenderer.send(PTY_RESIZE, sessionId, cols, rows),
  ptyDetach: (sessionId) => ipcRenderer.send(PTY_DETACH, sessionId),
  onPtyOutput: (listener) => {
    const handler = (_event: unknown, sessionId: string, data: Uint8Array) => listener(sessionId, data);
    ipcRenderer.on(PTY_OUTPUT, handler);
    return () => {
      ipcRenderer.removeListener(PTY_OUTPUT, handler);
    };
  },
  onPtyClosed: (listener) => {
    const handler = (_event: unknown, closed: PtyClosed) => listener(closed);
    ipcRenderer.on(PTY_CLOSED, handler);
    return () => {
      ipcRenderer.removeListener(PTY_CLOSED, handler);
    };
  },
};

contextBridge.exposeInMainWorld('calango', bridge);
