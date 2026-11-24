import { useEffect, useState } from 'react';
import type { DaemonConnection, DaemonState } from '../shared/ipc';
import { bridge } from './bridge';

export interface DaemonView {
  state: DaemonState;
  connection: DaemonConnection | null;
}

// useDaemon follows the main process supervisor. The connection (token included) is
// fetched again every time the daemon reaches running, so a restart never reuses a stale one.
export function useDaemon(): DaemonView {
  const [state, setState] = useState<DaemonState>({ status: 'starting' });
  const [connection, setConnection] = useState<DaemonConnection | null>(null);

  useEffect(() => {
    let active = true;
    const unsubscribe = bridge.onState((next) => {
      if (active) {
        setState(next);
      }
    });
    void bridge.getState().then((next) => {
      if (active) {
        setState(next);
      }
    });
    return () => {
      active = false;
      unsubscribe();
    };
  }, []);

  const running = state.status === 'running';
  useEffect(() => {
    if (!running) {
      setConnection(null);
      return;
    }
    let active = true;
    bridge.getConnection().then(
      (next) => {
        if (active) {
          setConnection(next);
        }
      },
      () => {
        if (active) {
          setConnection(null);
        }
      },
    );
    return () => {
      active = false;
    };
  }, [running]);

  return { state, connection };
}
