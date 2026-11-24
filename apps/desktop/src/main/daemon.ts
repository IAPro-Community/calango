import { execFile, spawn, type ChildProcess } from 'node:child_process';
import path from 'node:path';
import { promisify } from 'node:util';
import type { DaemonConnection, DaemonState } from '../shared/ipc';

const execFileAsync = promisify(execFile);

const BACKOFF_START_MS = 250;
const BACKOFF_CAP_MS = 4_000;
// A daemon that stayed up this long resets the backoff.
const STABLE_MS = 30_000;
const HEALTH_TIMEOUT_MS = 10_000;
const HEALTH_POLL_MS = 200;
const STOP_KILL_MS = 5_000;

export interface DaemonOptions {
  binary: string;
  /** Loopback host:port the daemon listens on. */
  addr: string;
  onState: (state: DaemonState) => void;
}

// readToken asks the binary for the local API token. The token leaves the process only
// through the return value: it is never written to disk or to logs.
export async function readToken(binary: string): Promise<string> {
  const { stdout } = await execFileAsync(binary, ['print-token'], { timeout: 5_000, maxBuffer: 4_096 });
  const token = stdout.trim();
  if (token === '') {
    throw new Error('print-token returned an empty token');
  }
  return token;
}

// Daemon runs the calango binary as a child process, waits until it answers /healthz,
// and restarts it with capped backoff when it exits.
export class Daemon {
  private child: ChildProcess | null = null;
  private connection: DaemonConnection | null = null;
  private state: DaemonState = { status: 'stopped' };
  private stopping = false;
  private backoffMs = BACKOFF_START_MS;
  private restartTimer: NodeJS.Timeout | null = null;
  private startedAt = 0;

  constructor(private readonly opts: DaemonOptions) {}

  getState(): DaemonState {
    return this.state;
  }

  connectionOrThrow(): DaemonConnection {
    if (this.connection === null) {
      throw new Error('daemon is not running');
    }
    return this.connection;
  }

  start(): Promise<void> {
    this.stopping = false;
    return this.launch();
  }

  async stop(): Promise<void> {
    this.stopping = true;
    if (this.restartTimer !== null) {
      clearTimeout(this.restartTimer);
      this.restartTimer = null;
    }
    const child = this.child;
    if (child === null) {
      this.setState({ status: 'stopped' });
      return;
    }
    const exited = Promise.withResolvers<void>();
    if (child.exitCode !== null || child.signalCode !== null) {
      exited.resolve();
    } else {
      const force = setTimeout(() => child.kill('SIGKILL'), STOP_KILL_MS);
      child.once('exit', () => {
        clearTimeout(force);
        exited.resolve();
      });
      child.kill('SIGTERM');
    }
    await exited.promise;
  }

  private async launch(): Promise<void> {
    this.setState({ status: 'starting' });
    let token: string;
    try {
      token = await readToken(this.opts.binary);
    } catch (err) {
      this.scheduleRestart(`token: ${err instanceof Error ? err.message : String(err)}`);
      return;
    }
    if (this.stopping) {
      return;
    }
    const baseUrl = `http://${this.opts.addr}`;

    // npm run prepends node_modules/.bin entries; they can shadow the user's authenticated CLI.
    const daemonPath = (process.env.PATH ?? '')
      .split(path.delimiter)
      .filter((entry) => !path.normalize(entry).toLowerCase().endsWith(`${path.sep}node_modules${path.sep}.bin`))
      .join(path.delimiter);
    const child = spawn(this.opts.binary, [], {
      env: { ...process.env, CALANGO_ADDR: this.opts.addr, PATH: daemonPath },
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    this.child = child;
    child.stdout?.pipe(process.stdout);
    child.stderr?.pipe(process.stderr);

    let gone = false;
    const onGone = (reason: string): void => {
      if (gone) {
        return;
      }
      gone = true;
      if (this.child !== child) {
        return;
      }
      this.child = null;
      this.connection = null;
      this.scheduleRestart(reason);
    };
    child.once('error', (err) => onGone(`spawn: ${err.message}`));
    child.once('exit', (code, signal) => onGone(`exited (code ${code ?? 'none'}, signal ${signal ?? 'none'})`));

    const healthy = await waitHealthy(baseUrl, () => this.child === child);
    if (!healthy) {
      if (this.child === child) {
        child.kill('SIGKILL');
      }
      return;
    }
    if (this.child !== child) {
      return;
    }
    this.connection = { baseUrl, token };
    this.startedAt = Date.now();
    this.setState({ status: 'running' });
  }

  private scheduleRestart(reason: string): void {
    if (this.stopping) {
      this.setState({ status: 'stopped' });
      return;
    }
    if (Date.now() - this.startedAt >= STABLE_MS) {
      this.backoffMs = BACKOFF_START_MS;
    }
    const wait = this.backoffMs;
    this.backoffMs = Math.min(this.backoffMs * 2, BACKOFF_CAP_MS);
    this.setState({ status: 'restarting', detail: `${reason}; retrying in ${wait} ms` });
    this.restartTimer = setTimeout(() => {
      this.restartTimer = null;
      void this.launch();
    }, wait);
  }

  private setState(state: DaemonState): void {
    this.state = state;
    this.opts.onState(state);
  }
}

// waitHealthy polls /healthz until it answers, the child stops being the current one,
// or the deadline passes.
async function waitHealthy(baseUrl: string, stillRunning: () => boolean): Promise<boolean> {
  const deadline = Date.now() + HEALTH_TIMEOUT_MS;
  while (stillRunning() && Date.now() < deadline) {
    try {
      const res = await fetch(`${baseUrl}/healthz`, { signal: AbortSignal.timeout(1_000) });
      if (res.ok) {
        return true;
      }
    } catch {
      // Not listening yet.
    }
    await delay(HEALTH_POLL_MS);
  }
  return false;
}

function delay(ms: number): Promise<void> {
  const { promise, resolve } = Promise.withResolvers<void>();
  setTimeout(resolve, ms);
  return promise;
}
