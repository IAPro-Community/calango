// Project preview in the internal browser (F4-T06). Pure rules plus the loopback probe.
// Nothing here imports Electron, so the main-process tests load this module under Node.
import { createHmac } from 'node:crypto';
import http from 'node:http';
import { PREVIEW_OWNER, type DevServer } from '../../shared/ipc';

/**
 * Owner key of preview tabs inside BrowserManager. The "@" can never appear in an agent id
 * (see AGENT_ID_PATTERN), so a preview can never share tabs or a profile with an agent.
 */
export { PREVIEW_OWNER };

/** Prefix of every preview partition, separate from the agent partitions. */
export const PREVIEW_PARTITION_PREFIX = 'persist:calango-preview-';

/**
 * Ports probed by "Detect local servers": the defaults of the common dev servers
 * (Next/Express/Rails 3000-3001, Vite preview 4173, Angular 4200, Flask 5000, Vite
 * 5173-5174, Django/Python 8000, generic 8080, Jupyter 8888). Kept short on purpose:
 * this is not a port scan.
 */
export const DEFAULT_PREVIEW_PORTS: readonly number[] = [3000, 3001, 4173, 4200, 5000, 5173, 5174, 8000, 8080, 8888];

export const MAX_PROBE_PORTS = 32;
export const DEFAULT_PROBE_TIMEOUT_MS = 800;
const MAX_PROBE_BODY_BYTES = 64 * 1024;
const MAX_TITLE_LENGTH = 120;
const PROJECT_ID_PATTERN = /^[A-Za-z0-9_-]{1,128}$/;

/**
 * Hosts a preview may use, compared after WHATWG URL parsing (which lowercases and turns
 * numeric forms such as 0x7f.1 into 127.0.0.1). No DNS name other than "localhost" is
 * accepted: Chromium resolves "localhost" to loopback itself, while names like
 * "localtest.me" or "localhost.evil.com" would go through DNS and could point anywhere.
 */
const LOOPBACK_HOSTS = new Set(['localhost', '127.0.0.1', '[::1]']);

/** Literal addresses the probe connects to; it never resolves a name. */
const PROBE_ADDRESSES: readonly { address: string; family: 4 | 6; host: string }[] = [
  { address: '127.0.0.1', family: 4, host: '127.0.0.1' },
  { address: '::1', family: 6, host: '[::1]' },
];

export interface PreviewTarget {
  /** Canonical origin, e.g. http://127.0.0.1:5173 */
  origin: string;
  /** First URL to load (same origin, path kept). */
  url: string;
  port: number;
}

export function parsePort(value: unknown): number | null {
  const n = typeof value === 'string' && /^\d{1,5}$/.test(value.trim()) ? Number(value.trim()) : value;
  return typeof n === 'number' && Number.isInteger(n) && n >= 1 && n <= 65535 ? n : null;
}

function effectivePort(url: URL): number {
  return url.port === '' ? 80 : Number(url.port);
}

/** loopbackHttpUrl parses a URL and accepts it only as plain http on a loopback host. */
function loopbackHttpUrl(raw: string): URL | null {
  if (raw.length === 0 || raw.length > 8192) return null;
  let url: URL;
  try {
    url = new URL(raw);
  } catch {
    return null;
  }
  if (url.protocol !== 'http:' || url.username !== '' || url.password !== '') return null;
  if (!LOOPBACK_HOSTS.has(url.hostname) || effectivePort(url) < 1) return null;
  return url;
}

/**
 * parsePreviewTarget accepts a port (5173 or "5173"), "localhost:5173", or a full
 * http://localhost|127.0.0.1|[::1]:PORT/path URL. Anything else (0.0.0.0, LAN or public
 * IPs, other hostnames, https, userinfo, other schemes) is refused with null.
 */
export function parsePreviewTarget(input: unknown): PreviewTarget | null {
  const port = parsePort(input);
  if (port !== null) {
    const origin = `http://127.0.0.1:${port}`;
    return { origin, url: `${origin}/`, port };
  }
  if (typeof input !== 'string') return null;
  const text = input.trim();
  if (text.length === 0 || /\s/.test(text)) return null;
  const candidate = /^[a-z][a-z0-9+.-]*:\/\//i.test(text) ? text : `http://${text}`;
  const url = loopbackHttpUrl(candidate);
  if (url === null) return null;
  const origin = `http://${url.hostname}:${effectivePort(url)}`;
  return { origin, url: `${origin}${url.pathname}${url.search}${url.hash}`, port: effectivePort(url) };
}

/**
 * isWithinPreviewOrigin is the containment rule of a preview tab: plain http, no userinfo,
 * the same port as the chosen origin, and a loopback host. localhost, 127.0.0.1 and [::1]
 * on the same port count as the same preview (dev servers mix them in links); any other
 * host or port is outside.
 */
export function isWithinPreviewOrigin(raw: unknown, origin: string): boolean {
  if (typeof raw !== 'string') return false;
  const url = loopbackHttpUrl(raw);
  const base = loopbackHttpUrl(origin);
  return url !== null && base !== null && effectivePort(url) === effectivePort(base);
}

export function isProjectId(value: unknown): value is string {
  return typeof value === 'string' && PROJECT_ID_PATTERN.test(value);
}

/**
 * partitionForPreview names the preview profile. Without a project (today: there is no
 * project registry yet, #82) every preview shares one profile; with a project id (hook for
 * #82/#83) each project gets its own. The name is an HMAC like the agent partitions.
 */
export function partitionForPreview(projectId: string | null, profileKey: Uint8Array): string {
  if (projectId !== null && !isProjectId(projectId)) throw new Error('invalid project id');
  if (profileKey.byteLength < 32) throw new Error('browser profile key is too short');
  const digest = createHmac('sha256', profileKey)
    .update(`calango-browser-profile:v1:preview:${projectId ?? ''}`)
    .digest('hex');
  return `${PREVIEW_PARTITION_PREFIX}${digest.slice(0, 32)}`;
}

/** parsePortList validates a port list from the renderer or the agent (short, unique). */
export function parsePortList(value: unknown): number[] | null {
  if (value === undefined || value === null) return [...DEFAULT_PREVIEW_PORTS];
  if (!Array.isArray(value) || value.length === 0 || value.length > MAX_PROBE_PORTS) return null;
  const ports = value.map(parsePort);
  if (ports.some((port) => port === null)) return null;
  return [...new Set(ports as number[])];
}

const ENTITIES: Record<string, string> = { amp: '&', lt: '<', gt: '>', quot: '"', apos: "'", '#39': "'" };

// C0/C1 controls and zero-width or bidi-override marks: shown as a space, never kept.
function isHiddenChar(code: number): boolean {
  return (
    code <= 0x1f ||
    (code >= 0x7f && code <= 0x9f) ||
    (code >= 0x200b && code <= 0x200f) ||
    (code >= 0x202a && code <= 0x202e) ||
    (code >= 0x2066 && code <= 0x2069)
  );
}

/** cleanTitle turns a page <title> into short plain text (it is page data, never markup). */
export function cleanTitle(raw: string): string {
  const decoded = raw.replace(/&(amp|lt|gt|quot|apos|#39);/g, (_m, name: string) => ENTITIES[name] ?? '');
  return Array.from(decoded, (char) => (isHiddenChar(char.codePointAt(0) ?? 0) ? ' ' : char))
    .join('')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, MAX_TITLE_LENGTH);
}

function guessKind(headers: http.IncomingHttpHeaders, body: string): string {
  const poweredBy = String(headers['x-powered-by'] ?? '');
  const server = String(headers.server ?? '');
  if (body.includes('/@vite/client')) return 'vite';
  if (/next\.js/i.test(poweredBy) || body.includes('/_next/')) return 'next';
  if (body.includes('ng-version') || body.includes('<app-root')) return 'angular';
  if (/webpack/i.test(body)) return 'webpack';
  if (/express/i.test(poweredBy)) return 'express';
  if (/python|werkzeug|uvicorn|gunicorn/i.test(server)) return 'python';
  return 'http';
}

export interface ProbeOptions {
  timeoutMs?: number;
}

function probeAddress(
  port: number,
  target: (typeof PROBE_ADDRESSES)[number],
  timeoutMs: number,
): Promise<DevServer | null> {
  return new Promise((resolve) => {
    let settled = false;
    const finish = (result: DevServer | null) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      request.destroy();
      resolve(result);
    };
    const request = http.request({
      host: target.address,
      family: target.family,
      port,
      method: 'GET',
      path: '/',
      agent: false,
      // Dev servers with host checks (Vite, webpack) accept "localhost".
      headers: { Host: `localhost:${port}`, Accept: 'text/html,*/*;q=0.5', 'User-Agent': 'Calango-preview-probe' },
    });
    const timer = setTimeout(() => finish(null), timeoutMs);
    request.on('error', () => finish(null));
    request.on('response', (response) => {
      const chunks: Buffer[] = [];
      let size = 0;
      const done = () => {
        const body = Buffer.concat(chunks).toString('utf8');
        const match = /<title[^>]*>([\s\S]*?)<\/title>/i.exec(body);
        finish({
          port,
          origin: `http://${target.host}:${port}`,
          status: response.statusCode ?? 0,
          title: match === null ? '' : cleanTitle(match[1] ?? ''),
          kind: guessKind(response.headers, body),
        });
      };
      response.on('data', (chunk: Buffer) => {
        size += chunk.byteLength;
        chunks.push(chunk);
        if (size >= MAX_PROBE_BODY_BYTES) done();
      });
      response.on('end', done);
      response.on('error', done);
    });
    request.end();
  });
}

/**
 * probePort asks one loopback port for "/" over HTTP: first 127.0.0.1, then ::1 (Vite on
 * recent Node may listen on ::1 only). It connects to literal loopback addresses only,
 * never resolves a name, does not follow redirects and gives up after timeoutMs.
 */
export async function probePort(port: number, options: ProbeOptions = {}): Promise<DevServer | null> {
  if (parsePort(port) === null) return null;
  const timeoutMs = Math.min(Math.max(options.timeoutMs ?? DEFAULT_PROBE_TIMEOUT_MS, 50), 5_000);
  for (const target of PROBE_ADDRESSES) {
    const found = await probeAddress(port, target, timeoutMs);
    if (found !== null) return found;
  }
  return null;
}

export interface DetectOptions extends ProbeOptions {
  /** Ports never reported or opened, such as the Calango daemon's own port. */
  exclude?: readonly number[];
}

/** detectDevServers probes a short list of loopback ports in parallel. */
export async function detectDevServers(ports: readonly number[], options: DetectOptions = {}): Promise<DevServer[]> {
  if (ports.length > MAX_PROBE_PORTS) throw new Error('too many ports to probe');
  const excluded = new Set(options.exclude ?? []);
  const wanted = [...new Set(ports)].filter((port) => parsePort(port) !== null && !excluded.has(port));
  const results = await Promise.all(wanted.map((port) => probePort(port, options)));
  return results.filter((result): result is DevServer => result !== null).sort((a, b) => a.port - b.port);
}
