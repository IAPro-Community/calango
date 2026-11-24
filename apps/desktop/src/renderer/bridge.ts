import type { CalangoBridge } from '../shared/ipc';

declare global {
  interface Window {
    calango: CalangoBridge;
  }
}

export const bridge: CalangoBridge = window.calango;
