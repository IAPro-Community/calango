import type { DaemonConnection, PtyClosed } from '../shared/ipc';

// Matches pty.ValidID in the daemon: 256 random bits, URL-safe base64 without padding.
const SESSION_ID = /^[A-Za-z0-9_-]{43}$/;
// The daemon refuses client frames above 64 KiB; a UTF-16 string of this length never
// encodes to more than three times as many bytes, so stay well under it.
const MAX_INPUT_CHARS = 16 * 1024;
const MAX_SIZE = 1000;

export function isSessionId(value: unknown): value is string {
  return typeof value === 'string' && SESSION_ID.test(value);
}

export function isTerminalSize(cols: unknown, rows: unknown): cols is number {
  return (
    Number.isInteger(cols) &&
    Number.isInteger(rows) &&
    (cols as number) >= 1 &&
    (rows as number) >= 1 &&
    (cols as number) <= MAX_SIZE &&
    (rows as number) <= MAX_SIZE
  );
}

export function isTerminalInput(data: unknown): data is string {
  return typeof data === 'string' && data.length > 0 && data.length <= MAX_INPUT_CHARS;
}

/** ptySocketUrl turns the daemon base URL into the WebSocket address of one session. */
export function ptySocketUrl(baseUrl: string, sessionId: string): string {
  const url = new URL(baseUrl);
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new Error('daemon base URL must be http or https');
  }
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  url.pathname = `/v1/pty/${sessionId}`;
  url.search = '';
  return url.toString();
}

/** parseExit reads the daemon's {"type":"exit"} control frame; anything else is null. */
export function parseExit(text: string): { reason?: string; exitCode?: number } | null {
  try {
    const frame: unknown = JSON.parse(text);
    if (typeof frame !== 'object' || frame === null || (frame as { type?: unknown }).type !== 'exit') {
      return null;
    }
    const { reason, exit_code: exitCode } = frame as { reason?: unknown; exit_code?: unknown };
    return {
      reason: typeof reason === 'string' ? reason : undefined,
      exitCode: Number.isInteger(exitCode) ? (exitCode as number) : undefined,
    };
  } catch {
    return null;
  }
}

export interface PtyBridgeOptions {
  connection(): DaemonConnection;
  /** Delivers output to the renderer that attached the session. */
  sendOutput(ownerId: number, sessionId: string, data: Uint8Array): void;
  sendClosed(ownerId: number, closed: PtyClosed): void;
}

interface Attached {
  ownerId: number;
  socket: WebSocket;
  ended: boolean;
  /** Closed on purpose by detach; the renderer already knows. */
  detached: boolean;
  reason?: string;
  exitCode?: number;
}

/**
 * PtyBridge keeps the terminal WebSockets in the main process. Node's WebSocket accepts
 * an Authorization header, which a renderer WebSocket cannot send; the token therefore
 * stays out of the URL (ADR 0015). Each socket belongs to the webContents that opened it.
 */
export class PtyBridge {
  private readonly sockets = new Map<string, Attached>();
  private readonly encoder = new TextEncoder();

  constructor(private readonly options: PtyBridgeOptions) {}

  attach(ownerId: number, sessionId: string): Promise<void> {
    if (!isSessionId(sessionId)) {
      return Promise.reject(new Error('invalid terminal session id'));
    }
    this.detach(sessionId);
    const { baseUrl, token } = this.options.connection();
    const socket = new WebSocket(ptySocketUrl(baseUrl, sessionId), {
      headers: { Authorization: `Bearer ${token}` },
    });
    socket.binaryType = 'arraybuffer';
    const entry: Attached = { ownerId, socket, ended: false, detached: false };
    this.sockets.set(sessionId, entry);

    return new Promise((resolve, reject) => {
      let opened = false;
      socket.addEventListener('open', () => {
        opened = true;
        resolve();
      });
      socket.addEventListener('message', (event: MessageEvent) => {
        if (event.data instanceof ArrayBuffer) {
          this.options.sendOutput(ownerId, sessionId, new Uint8Array(event.data));
          return;
        }
        if (typeof event.data === 'string') {
          const exit = parseExit(event.data);
          if (exit !== null) {
            entry.ended = true;
            entry.reason = exit.reason;
            entry.exitCode = exit.exitCode;
          }
        }
      });
      socket.addEventListener('close', () => {
        if (this.sockets.get(sessionId) === entry) {
          this.sockets.delete(sessionId);
        }
        if (!opened) {
          // The daemon answered the upgrade with an HTTP error (not found, busy, auth).
          reject(new Error('could not attach to the terminal session'));
          return;
        }
        if (entry.detached) {
          return;
        }
        this.options.sendClosed(ownerId, {
          sessionId,
          ended: entry.ended,
          reason: entry.reason,
          exitCode: entry.exitCode,
        });
      });
    });
  }

  input(ownerId: number, sessionId: string, data: string): void {
    const entry = this.owned(ownerId, sessionId);
    if (entry !== undefined && isTerminalInput(data) && entry.socket.readyState === WebSocket.OPEN) {
      entry.socket.send(this.encoder.encode(data));
    }
  }

  resize(ownerId: number, sessionId: string, cols: number, rows: number): void {
    const entry = this.owned(ownerId, sessionId);
    if (entry !== undefined && isTerminalSize(cols, rows) && entry.socket.readyState === WebSocket.OPEN) {
      entry.socket.send(JSON.stringify({ type: 'resize', cols, rows }));
    }
  }

  /** detach closes the socket only; the session keeps running on the daemon. */
  detach(sessionId: string, ownerId?: number): void {
    const entry = this.sockets.get(sessionId);
    if (entry === undefined || (ownerId !== undefined && entry.ownerId !== ownerId)) {
      return;
    }
    this.sockets.delete(sessionId);
    entry.detached = true;
    entry.socket.close(1000);
  }

  /** detachOwner closes every socket of a renderer that went away. */
  detachOwner(ownerId: number): void {
    for (const [sessionId, entry] of this.sockets) {
      if (entry.ownerId === ownerId) {
        this.detach(sessionId);
      }
    }
  }

  detachAll(): void {
    for (const sessionId of [...this.sockets.keys()]) {
      this.detach(sessionId);
    }
  }

  private owned(ownerId: number, sessionId: string): Attached | undefined {
    if (!isSessionId(sessionId)) {
      return undefined;
    }
    const entry = this.sockets.get(sessionId);
    return entry?.ownerId === ownerId ? entry : undefined;
  }
}
