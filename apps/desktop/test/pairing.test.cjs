const assert = require('node:assert/strict');
const { test } = require('node:test');
const {
  encodePairingPayload,
  formatCountdown,
  formatPairingCode,
  nodeLabel,
  normalizePairingCode,
  PairingPayloadError,
  parsePairingPayload,
  secondsLeft,
} = require('../dist/shared/pairing.js');
const { QR_QUIET_ZONE, qrMatrix } = require('../dist/shared/qr.js');

const base = { code: 'ABCD2345', scope: 'approve_only', expiresAt: 1764000300 };

function reason(fn) {
  try {
    fn();
  } catch (err) {
    assert.ok(err instanceof PairingPayloadError, `unexpected error ${err}`);
    return `${err.reason}:${err.field ?? ''}`;
  }
  assert.fail('expected a PairingPayloadError');
}

test('payload round-trips with and without optional fields', () => {
  const minimal = encodePairingPayload(base);
  assert.equal(minimal, 'calango://pair?v=1&code=ABCD2345&scope=approve_only&exp=1764000300');
  assert.deepEqual(parsePairingPayload(minimal), base);

  const full = {
    ...base,
    scope: 'full',
    node: 'Mesa do Kyo & cia',
    addr: '100.64.0.1:7317',
    fingerprint: 'sha256:AbC',
  };
  assert.deepEqual(parsePairingPayload(encodePairingPayload(full)), full);
});

test('payload never carries anything but its documented fields', () => {
  const uri = encodePairingPayload({ ...base, token: 'secret-token' });
  assert.doesNotMatch(uri, /secret-token|token=/);
  assert.deepEqual([...new URL(uri).searchParams.keys()], ['v', 'code', 'scope', 'exp']);
});

test('parse refuses unknown versions and foreign URIs', () => {
  assert.equal(
    reason(() => parsePairingPayload('calango://pair?v=2&code=ABCD2345&scope=read&exp=1')),
    'unsupported_version:v',
  );
  assert.equal(
    reason(() => parsePairingPayload('calango://pair?code=ABCD2345&scope=read&exp=1')),
    'missing_field:v',
  );
  assert.equal(
    reason(() => parsePairingPayload('https://example.com/pair?v=1')),
    'not_pairing_uri:',
  );
  assert.equal(
    reason(() => parsePairingPayload('calango://pairing?v=1&code=ABCD2345')),
    'not_pairing_uri:',
  );
});

test('parse reports missing and invalid fields', () => {
  assert.equal(
    reason(() => parsePairingPayload('calango://pair?v=1&scope=read&exp=1')),
    'missing_field:code',
  );
  assert.equal(
    reason(() => parsePairingPayload('calango://pair?v=1&code=ABCD2345&exp=1')),
    'missing_field:scope',
  );
  assert.equal(
    reason(() => parsePairingPayload('calango://pair?v=1&code=ABCD2345&scope=read')),
    'missing_field:exp',
  );
  assert.equal(
    reason(() => parsePairingPayload('calango://pair?v=1&code=ABCD1234&scope=read&exp=1')),
    'invalid_field:code',
  );
  assert.equal(
    reason(() => parsePairingPayload('calango://pair?v=1&code=ABCD2345&scope=admin&exp=1')),
    'invalid_field:scope',
  );
  assert.equal(
    reason(() => parsePairingPayload('calango://pair?v=1&code=ABCD2345&scope=read&exp=-5')),
    'invalid_field:exp',
  );
  assert.equal(
    reason(() => parsePairingPayload('calango://pair?v=1&code=ABCD2345&scope=read&exp=1&addr=a%20b')),
    'invalid_field:addr',
  );
});

test('parse ignores unknown v1 parameters and normalizes the typed code form', () => {
  const parsed = parsePairingPayload(' calango://pair?v=1&code=abcd-2345&scope=read&exp=10&future=x ');
  assert.deepEqual(parsed, { code: 'ABCD2345', scope: 'read', expiresAt: 10 });
});

test('encode rejects invalid payloads', () => {
  assert.equal(
    reason(() => encodePairingPayload({ ...base, code: 'short' })),
    'invalid_field:code',
  );
  assert.equal(
    reason(() => encodePairingPayload({ ...base, scope: 'admin' })),
    'invalid_field:scope',
  );
  assert.equal(
    reason(() => encodePairingPayload({ ...base, expiresAt: 1.5 })),
    'invalid_field:exp',
  );
  assert.equal(
    reason(() => encodePairingPayload({ ...base, node: 'x'.repeat(65) })),
    'invalid_field:node',
  );
});

test('code formatting, countdown and node label helpers', () => {
  assert.equal(normalizePairingCode(' abcd-2345 '), 'ABCD2345');
  assert.equal(formatPairingCode('ABCD2345'), 'ABCD-2345');
  assert.equal(secondsLeft(10_000, 9_001), 1);
  assert.equal(secondsLeft(10_000, 10_000), 0);
  assert.equal(secondsLeft(10_000, 20_000), 0);
  assert.equal(formatCountdown(300), '5:00');
  assert.equal(formatCountdown(59), '0:59');
  assert.equal(nodeLabel('  meu-pc\n'), 'meu-pc');
  assert.equal(nodeLabel('\u0007'), undefined);
  assert.equal(nodeLabel('a'.repeat(100)).length, 64);
});

test('QR matrix is square, has a quiet zone and draws dark modules', () => {
  const qr = qrMatrix(encodePairingPayload({ ...base, node: 'meu-pc' }));
  assert.ok(qr.size >= 21 && (qr.size - 17) % 4 === 0, `unexpected size ${qr.size}`);
  assert.equal(qr.viewBox, qr.size + 2 * QR_QUIET_ZONE);
  // The top-left finder pattern starts at the quiet-zone offset.
  assert.ok(qr.path.startsWith(`M${QR_QUIET_ZONE} ${QR_QUIET_ZONE}h1v1h-1z`));
  assert.match(qr.path, /^(M\d+ \d+h1v1h-1z)+$/);
});
