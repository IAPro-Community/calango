// Pairing QR payload, version 1. The phone app (#67) reads this URI from the QR shown
// by the desktop and redeems the code with POST /v1/pair. The format is documented in
// DEV/SPECS/modules/pairing-auth.md; keep both in sync.
//
// The payload carries only what the phone needs to redeem a single-use code: never the
// local API token, a device token or any other long-lived secret.

export const PAIRING_URI_PREFIX = 'calango://pair';
export const PAIRING_PAYLOAD_VERSION = 1;

export const PAIRING_SCOPES = ['full', 'approve_only', 'read'] as const;
export type PairingScope = (typeof PAIRING_SCOPES)[number];

// Same alphabet and length as internal/core/device.go (no I, O, 0, 1).
const CODE_PATTERN = /^[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{8}$/;
const MAX_NODE_LENGTH = 64;
const MAX_ADDR_LENGTH = 255;
const MAX_FINGERPRINT_LENGTH = 128;
// Transport-defined values (#43) stay opaque but must be printable ASCII without spaces.
const OPAQUE_PATTERN = /^[\x21-\x7e]+$/;

export interface PairingPayload {
  /** Single-use pairing code, 8 characters, as issued by POST /v1/pairing-codes. */
  code: string;
  /** Scope the code grants; the app should check it against the POST /v1/pair response. */
  scope: PairingScope;
  /** Expiry as Unix seconds (UTC). */
  expiresAt: number;
  /** Display label of the node (for example the computer name). Not an identity. */
  node?: string;
  /** Node address on the transport. Its format is decided in #43; absent until then. */
  addr?: string;
  /** Fingerprint of the node's transport key, defined with #43; absent until then. */
  fingerprint?: string;
}

export type PairingParseError = 'not_pairing_uri' | 'unsupported_version' | 'missing_field' | 'invalid_field';

export class PairingPayloadError extends Error {
  constructor(
    readonly reason: PairingParseError,
    readonly field?: string,
  ) {
    super(field === undefined ? `pairing payload: ${reason}` : `pairing payload: ${reason} (${field})`);
    this.name = 'PairingPayloadError';
  }
}

export function isPairingScope(value: unknown): value is PairingScope {
  return typeof value === 'string' && (PAIRING_SCOPES as readonly string[]).includes(value);
}

/** normalizePairingCode mirrors core.NormalizePairingCode: drops spaces and dashes, upper-cases. */
export function normalizePairingCode(value: string): string {
  return value.replace(/[\s-]/g, '').toUpperCase();
}

/** formatPairingCode groups the code as XXXX-XXXX for reading aloud or typing. */
export function formatPairingCode(code: string): string {
  const normalized = normalizePairingCode(code);
  return normalized.length === 8 ? `${normalized.slice(0, 4)}-${normalized.slice(4)}` : normalized;
}

function checkPayload(payload: PairingPayload): void {
  if (!CODE_PATTERN.test(payload.code)) throw new PairingPayloadError('invalid_field', 'code');
  if (!isPairingScope(payload.scope)) throw new PairingPayloadError('invalid_field', 'scope');
  if (!Number.isSafeInteger(payload.expiresAt) || payload.expiresAt <= 0) {
    throw new PairingPayloadError('invalid_field', 'exp');
  }
  if (payload.node !== undefined && (payload.node.trim() === '' || payload.node.length > MAX_NODE_LENGTH)) {
    throw new PairingPayloadError('invalid_field', 'node');
  }
  if (payload.addr !== undefined && (payload.addr.length > MAX_ADDR_LENGTH || !OPAQUE_PATTERN.test(payload.addr))) {
    throw new PairingPayloadError('invalid_field', 'addr');
  }
  if (
    payload.fingerprint !== undefined &&
    (payload.fingerprint.length > MAX_FINGERPRINT_LENGTH || !OPAQUE_PATTERN.test(payload.fingerprint))
  ) {
    throw new PairingPayloadError('invalid_field', 'fp');
  }
}

/** encodePairingPayload builds the version 1 URI. Optional fields are omitted when absent. */
export function encodePairingPayload(payload: PairingPayload): string {
  checkPayload(payload);
  const params = new URLSearchParams();
  params.set('v', String(PAIRING_PAYLOAD_VERSION));
  params.set('code', payload.code);
  params.set('scope', payload.scope);
  params.set('exp', String(payload.expiresAt));
  if (payload.node !== undefined) params.set('node', payload.node);
  if (payload.addr !== undefined) params.set('addr', payload.addr);
  if (payload.fingerprint !== undefined) params.set('fp', payload.fingerprint);
  return `${PAIRING_URI_PREFIX}?${params.toString()}`;
}

/**
 * parsePairingPayload reads a version 1 URI. Unknown parameters are ignored so later
 * v1 additions stay readable; a different `v` is refused.
 */
export function parsePairingPayload(text: string): PairingPayload {
  const value = text.trim();
  if (!value.startsWith(`${PAIRING_URI_PREFIX}?`)) {
    throw new PairingPayloadError('not_pairing_uri');
  }
  const params = new URLSearchParams(value.slice(PAIRING_URI_PREFIX.length + 1));
  const version = params.get('v');
  if (version === null) throw new PairingPayloadError('missing_field', 'v');
  if (version !== String(PAIRING_PAYLOAD_VERSION)) throw new PairingPayloadError('unsupported_version', 'v');

  const required = (name: string): string => {
    const field = params.get(name);
    if (field === null || field === '') throw new PairingPayloadError('missing_field', name);
    return field;
  };
  const optional = (name: string): string | undefined => {
    const field = params.get(name);
    return field === null || field === '' ? undefined : field;
  };

  const code = normalizePairingCode(required('code'));
  const scope = required('scope');
  const exp = required('exp');
  if (!/^\d{1,15}$/.test(exp)) throw new PairingPayloadError('invalid_field', 'exp');
  if (!isPairingScope(scope)) throw new PairingPayloadError('invalid_field', 'scope');

  const payload: PairingPayload = { code, scope, expiresAt: Number(exp) };
  const node = optional('node');
  const addr = optional('addr');
  const fingerprint = optional('fp');
  if (node !== undefined) payload.node = node;
  if (addr !== undefined) payload.addr = addr;
  if (fingerprint !== undefined) payload.fingerprint = fingerprint;
  checkPayload(payload);
  return payload;
}

/** secondsLeft is the whole number of seconds until expiresAtMs, never negative. */
export function secondsLeft(expiresAtMs: number, nowMs: number): number {
  return Math.max(0, Math.ceil((expiresAtMs - nowMs) / 1000));
}

/** formatCountdown renders seconds as m:ss. */
export function formatCountdown(seconds: number): string {
  const safe = Math.max(0, Math.floor(seconds));
  return `${Math.floor(safe / 60)}:${String(safe % 60).padStart(2, '0')}`;
}

/** nodeLabel turns a host name into the optional `node` field, or undefined when unusable. */
export function nodeLabel(hostname: string): string | undefined {
  // Control characters are stripped so the label is safe to show on the phone.
  // biome-ignore lint/suspicious/noControlCharactersInRegex: the range is intentional.
  const cleaned = hostname.replace(/[\u0000-\u001f\u007f]/g, '').trim();
  if (cleaned === '') return undefined;
  return cleaned.slice(0, MAX_NODE_LENGTH).trim();
}
