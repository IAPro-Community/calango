const assert = require('node:assert/strict');
const test = require('node:test');
const { isSessionId, isTerminalInput, isTerminalSize, parseExit, ptySocketUrl } = require('../dist/main/pty.js');

const id = `${'A'.repeat(21)}-_${'b'.repeat(20)}`;

test('session ids match the daemon format only', () => {
  assert.equal(isSessionId(id), true);
  for (const bad of ['', 'abc', `${id}x`, `../${id.slice(3)}`, `${'A'.repeat(42)}/`, 42, null]) {
    assert.equal(isSessionId(bad), false, String(bad));
  }
});

test('socket url targets /v1/pty/{id} on the daemon and drops any query', () => {
  assert.equal(ptySocketUrl('http://127.0.0.1:7317', id), `ws://127.0.0.1:7317/v1/pty/${id}`);
  assert.equal(ptySocketUrl('https://node.example:443/base?x=1', id), `wss://node.example/v1/pty/${id}`);
  assert.throws(() => ptySocketUrl('file:///tmp', id));
});

test('terminal size and input bounds follow the daemon limits', () => {
  assert.equal(isTerminalSize(80, 24), true);
  assert.equal(isTerminalSize(1000, 1000), true);
  for (const [c, r] of [
    [0, 24],
    [80, 0],
    [1001, 24],
    [80.5, 24],
    ['80', 24],
  ]) {
    assert.equal(isTerminalSize(c, r), false, `${c}x${r}`);
  }
  assert.equal(isTerminalInput('ls\r'), true);
  assert.equal(isTerminalInput(''), false);
  assert.equal(isTerminalInput('x'.repeat(16 * 1024 + 1)), false);
  assert.equal(isTerminalInput(Buffer.from('x')), false);
});

test('exit frames are parsed; anything else is ignored', () => {
  assert.deepEqual(parseExit('{"type":"exit","reason":"exited","exit_code":3}'), { reason: 'exited', exitCode: 3 });
  assert.deepEqual(parseExit('{"type":"exit","reason":"terminated"}'), { reason: 'terminated', exitCode: undefined });
  assert.equal(parseExit('{"type":"resize"}'), null);
  assert.equal(parseExit('not json'), null);
  assert.equal(parseExit('null'), null);
});
