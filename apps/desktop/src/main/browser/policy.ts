// Pure rules of the internal browser (F5-T01). Nothing here imports Electron at runtime,
// so the main-process tests can load this module under plain Node.
import { createHmac } from 'node:crypto';

/** Prefix of every agent browser partition. `persist:` keeps cookies and storage on disk. */
export const AGENT_PARTITION_PREFIX = 'persist:calango-agent-';

const AGENT_ID_PATTERN = /^[A-Za-z0-9_-]{1,128}$/;
const PROFILE_KEY_BYTES = 32;
const ALLOWED_PROTOCOLS = new Set(['http:', 'https:']);
const LOCAL_HOSTS = new Set(['localhost', '127.0.0.1', '[::1]']);

export function isAgentId(value: unknown): value is string {
  return typeof value === 'string' && AGENT_ID_PATTERN.test(value);
}

export function assertAgentId(value: unknown): string {
  if (!isAgentId(value)) {
    throw new Error('invalid agent id');
  }
  return value;
}

/**
 * partitionForAgent derives the persistent partition of one agent. The name is an HMAC of
 * the agent id under a per-install secret, so it is stable for the same agent, different
 * for every other agent, and cannot be guessed from the agent id alone.
 */
export function partitionForAgent(agentId: string, profileKey: Uint8Array): string {
  assertAgentId(agentId);
  if (profileKey.byteLength < PROFILE_KEY_BYTES) {
    throw new Error('browser profile key is too short');
  }
  const digest = createHmac('sha256', profileKey).update(`calango-browser-profile:v1:agent:${agentId}`).digest('hex');
  return `${AGENT_PARTITION_PREFIX}${digest.slice(0, 32)}`;
}

/**
 * isNavigableUrl is the single protocol gate of the browser: only http and https, with no
 * embedded credentials (a common phishing disguise). Everything else (file:, javascript:,
 * data:, chrome:, devtools:, custom app schemes, mailto: ...) is refused.
 */
export function isNavigableUrl(raw: unknown): raw is string {
  if (typeof raw !== 'string' || raw.length === 0 || raw.length > 8192) {
    return false;
  }
  let url: URL;
  try {
    url = new URL(raw);
  } catch {
    return false;
  }
  return ALLOWED_PROTOCOLS.has(url.protocol) && url.hostname !== '' && url.username === '' && url.password === '';
}

/**
 * normalizeAddress turns what the person typed in the address bar into a navigable URL, or
 * null. Text with no scheme gets https (http for loopback hosts, so a local dev server
 * preview works). There is no search fallback: typed text is never sent to a search engine.
 */
export function normalizeAddress(input: unknown): string | null {
  if (typeof input !== 'string') {
    return null;
  }
  const text = input.trim();
  if (text.length === 0 || /\s/.test(text)) {
    return null;
  }
  const scheme = /^([a-z][a-z0-9+.-]*):(.*)$/i.exec(text);
  // "localhost:5173" looks like a scheme; a port after the colon means it is a host.
  const hasScheme = scheme !== null && !/^\d/.test(scheme[2] ?? '');
  let candidate = text;
  if (!hasScheme) {
    const host = text.split(/[/?#]/, 1)[0]?.replace(/:\d+$/, '').toLowerCase() ?? '';
    candidate = `${LOCAL_HOSTS.has(host) ? 'http' : 'https'}://${text}`;
  }
  if (!isNavigableUrl(candidate)) {
    return null;
  }
  return new URL(candidate).toString();
}

export type WindowOpenDecision = 'tab' | 'deny';

/** decideWindowOpen: popups never become free windows; a valid http(s) target becomes a tab. */
export function decideWindowOpen(url: unknown): WindowOpenDecision {
  return isNavigableUrl(url) ? 'tab' : 'deny';
}

/** Permission policy: every web permission is denied by default (camera, mic, location...). */
export function permissionAllowed(_permission: string): boolean {
  return false;
}

export interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

/** parseBounds validates a rectangle from the renderer and clamps it to the window. */
export function parseBounds(value: unknown, max: { width: number; height: number }): Rect | null {
  if (value === null || typeof value !== 'object') {
    return null;
  }
  const record = value as Record<string, unknown>;
  const numbers = ['x', 'y', 'width', 'height'].map((key) => record[key]);
  if (!numbers.every((n) => typeof n === 'number' && Number.isFinite(n))) {
    return null;
  }
  const [x, y, width, height] = (numbers as number[]).map((n) => Math.max(0, Math.round(n)));
  const cx = Math.min(x, max.width);
  const cy = Math.min(y, max.height);
  return { x: cx, y: cy, width: Math.min(width, max.width - cx), height: Math.min(height, max.height - cy) };
}
