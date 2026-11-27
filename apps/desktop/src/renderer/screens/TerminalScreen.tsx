import { useCallback, useEffect, useRef, useState } from 'react';
import { ApiError, type PtySession } from '@calango/sdk-ts';
import { FitAddon } from '@xterm/addon-fit';
import { Terminal } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import { bridge } from '../bridge';
import type { Translate } from '../i18n';
import type { ScreenProps } from './props';
import { Button, PageHeader } from '@';

type Status = 'connecting' | 'connected' | 'ended' | 'disconnected';

interface Ended {
  exitCode?: number;
}

// Terminal screen (F4-T03, ADE foundation). REST calls go through the SDK; the socket
// for the session lives in the main process and reaches this screen over IPC.
export function TerminalScreen({ client, t }: ScreenProps) {
  const [session, setSession] = useState<PtySession | null>(null);
  // True once the daemon reports the shown session ended; its last output stays visible.
  const [sessionEnded, setSessionEnded] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const describe = useCallback(
    (err: unknown) => {
      if (err instanceof ApiError && err.code === 'rate_limited') {
        return t('terminalLimit');
      }
      return err instanceof Error ? err.message : String(err);
    },
    [t],
  );

  // Reopen a session left running by an earlier visit to this screen.
  useEffect(() => {
    let active = true;
    client.listPtySessions().then(
      (sessions) => {
        if (active) {
          setSession(sessions.find((s) => !s.attached) ?? null);
          setLoading(false);
        }
      },
      (err: unknown) => {
        if (active) {
          setError(describe(err));
          setLoading(false);
        }
      },
    );
    return () => {
      active = false;
    };
  }, [client, describe]);

  async function openSession() {
    setError(null);
    try {
      const created = await client.createPtySession({});
      setSessionEnded(false);
      setSession(created);
    } catch (err) {
      setError(describe(err));
    }
  }

  async function endSession() {
    if (session === null || sessionEnded) {
      return;
    }
    setError(null);
    try {
      await client.deletePtySession(session.id);
    } catch (err) {
      if (!(err instanceof ApiError && err.status === 404)) {
        setError(describe(err));
        return;
      }
    }
    setSession(null);
  }

  return (
    <section className="screen terminal-screen">
      <PageHeader
        title={t('navTerminal')}
        actions={
          <>
            <Button variant="ghost" onClick={() => void openSession()} disabled={session !== null && !sessionEnded}>
              {t('terminalNew')}
            </Button>
            <Button variant="danger" onClick={() => void endSession()} disabled={session === null || sessionEnded}>
              {t('terminalEnd')}
            </Button>
          </>
        }
      />
      <p className="muted">{t('terminalIntro')}</p>
      {error && <p className="error">{error}</p>}
      {session === null ? (
        !loading && <p className="muted">{t('terminalEmpty')}</p>
      ) : (
        <TerminalView key={session.id} session={session} t={t} onEnded={() => setSessionEnded(true)} />
      )}
    </section>
  );
}

function cssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

function TerminalView({ session, t, onEnded }: { session: PtySession; t: Translate; onEnded: () => void }) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const [status, setStatus] = useState<Status>('connecting');
  const [ended, setEnded] = useState<Ended | null>(null);
  const onEndedRef = useRef(onEnded);
  onEndedRef.current = onEnded;

  useEffect(() => {
    const host = hostRef.current;
    if (host === null) {
      return;
    }
    const id = session.id;
    const mono = cssVar('--calango-font-mono');
    const term = new Terminal({
      cursorBlink: true,
      fontFamily: `${mono ? `"${mono}", ` : ''}ui-monospace, monospace`,
      fontSize: 13,
      scrollback: 5000,
      theme: {
        background: cssVar('--calango-color-night'),
        foreground: cssVar('--calango-color-paper'),
        cursor: cssVar('--calango-color-postit'),
        selectionBackground: cssVar('--calango-color-ink-soft'),
      },
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(host);

    let attached = false;
    const sendSize = () => {
      try {
        fit.fit();
      } catch {
        return;
      }
      if (attached) {
        bridge.ptyResize(id, term.cols, term.rows);
      }
    };
    sendSize();

    const offOutput = bridge.onPtyOutput((sessionId, data) => {
      if (sessionId === id) {
        term.write(data);
      }
    });
    const offClosed = bridge.onPtyClosed((closed) => {
      if (closed.sessionId !== id) {
        return;
      }
      attached = false;
      if (closed.ended) {
        setEnded({ exitCode: closed.exitCode });
        setStatus('ended');
        onEndedRef.current();
      } else {
        setStatus('disconnected');
      }
    });
    const input = term.onData((data) => bridge.ptyInput(id, data));
    const observer = new ResizeObserver(sendSize);
    observer.observe(host);

    let disposed = false;
    bridge.ptyAttach(id).then(
      () => {
        if (disposed) {
          bridge.ptyDetach(id);
          return;
        }
        attached = true;
        setStatus('connected');
        sendSize();
        term.focus();
      },
      () => {
        if (!disposed) {
          setStatus('disconnected');
        }
      },
    );

    return () => {
      disposed = true;
      observer.disconnect();
      input.dispose();
      offOutput();
      offClosed();
      bridge.ptyDetach(id);
      term.dispose();
    };
  }, [session.id]);

  return (
    <div className="terminal-wrap">
      <p className="muted terminal-status" aria-live="polite">
        {status === 'connecting' && t('terminalConnecting')}
        {status === 'connected' && `${t('terminalConnected')} · ${session.cwd}`}
        {status === 'disconnected' && t('terminalDisconnected')}
        {status === 'ended' && (
          <>
            {t('terminalEnded')}
            {ended?.exitCode !== undefined && ` ${t('terminalExitCode')}: ${ended.exitCode}`}
          </>
        )}
      </p>
      <div className="terminal-host" ref={hostRef} />
    </div>
  );
}
