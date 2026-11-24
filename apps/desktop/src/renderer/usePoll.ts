import { useCallback, useEffect, useState } from 'react';

export interface Poll<T> {
  data: T | null;
  error: string | null;
  refresh: () => void;
}

// usePoll reloads data every intervalMs. load must be stable (useCallback) because a
// new load function restarts the loop.
export function usePoll<T>(load: () => Promise<T>, intervalMs: number): Poll<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let active = true;
    let timer: number | undefined;
    const run = async (): Promise<void> => {
      try {
        const next = await load();
        if (active) {
          setData(next);
          setError(null);
        }
      } catch (err) {
        if (active) {
          setError(err instanceof Error ? err.message : String(err));
        }
      }
      if (active) {
        timer = window.setTimeout(() => void run(), intervalMs);
      }
    };
    void run();
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [load, intervalMs, tick]);

  const refresh = useCallback(() => setTick((n) => n + 1), []);
  return { data, error, refresh };
}
