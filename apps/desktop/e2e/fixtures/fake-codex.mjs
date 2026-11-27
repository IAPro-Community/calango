#!/usr/bin/env node
// Deterministic stand-in for the Codex CLI in the desktop E2E tests. It answers the
// two commands the daemon runs (internal/harness/codex.go) and never reaches a model:
//   codex login status         -> always signed in
//   codex exec ... -o <file> - -> reads the prompt on stdin, writes a fixed reply
// The last prompt line with an "[e2e:...]" marker picks the behaviour, so each test
// uses its own conversation:
//   contains "[e2e:slow]" -> waits until it is killed (used to test cancellation)
//   contains "[e2e:tool]" -> calls the granted create_note tool through the turn's MCP
//                            endpoint, waits for the approval and reports the result
//   anything else         -> replies FAKE_REPLY
import { readFileSync, writeFileSync } from 'node:fs';

const FAKE_REPLY = 'Resposta determinística do provider fake.';

const args = process.argv.slice(2);

if (args[0] === 'login' && args[1] === 'status') {
  process.stdout.write('Logged in using ChatGPT (fake)\n');
  process.exit(0);
}

if (args[0] !== 'exec') {
  process.stderr.write(`fake codex: unsupported command ${args[0] ?? '(none)'}\n`);
  process.exit(2);
}

const useJson = args.includes('--json');
const outPath = valueAfter('-o');
if (!useJson && outPath === undefined) {
  process.stderr.write('fake codex: -o or --json is required\n');
  process.exit(2);
}
const prompt = readFileSync(0, 'utf8');
const lastLine = prompt.split('\n').findLast((line) => line.includes('[e2e:')) ?? '';

function emitReply(text) {
  if (useJson) {
    process.stdout.write(`${JSON.stringify({ type: 'thread.started', thread_id: 'e2e' })}\n`);
    process.stdout.write(
      `${JSON.stringify({ type: 'item.completed', item: { id: 'msg-1', type: 'agent_message', text } })}\n`,
    );
    process.stdout.write(
      `${JSON.stringify({ type: 'turn.completed', usage: { input_tokens: 7, output_tokens: 3 } })}\n`,
    );
    return;
  }
  if (outPath !== undefined) {
    writeFileSync(outPath, text);
  }
}

if (lastLine.includes('[e2e:slow]')) {
  // Stays busy until the daemon stops the run and kills this process.
  setInterval(() => {}, 1_000);
} else if (lastLine.includes('[e2e:tool]')) {
  const result = await callNoteTool();
  emitReply(`Ferramenta respondeu: ${result}`);
} else {
  emitReply(FAKE_REPLY);
}

function valueAfter(flag) {
  const i = args.indexOf(flag);
  return i === -1 ? undefined : args[i + 1];
}

// configValue reads `-c mcp_servers.calango.<key>="<value>"` overrides.
function configValue(key) {
  for (let i = 0; i < args.length - 1; i++) {
    if (args[i] === '-c' && args[i + 1].startsWith(`mcp_servers.calango.${key}=`)) {
      return JSON.parse(args[i + 1].slice(`mcp_servers.calango.${key}=`.length));
    }
  }
  return undefined;
}

async function callNoteTool() {
  const url = configValue('url');
  const tokenEnv = configValue('bearer_token_env_var');
  if (url === undefined || tokenEnv === undefined) {
    return 'sem endpoint MCP';
  }
  const headers = {
    'Content-Type': 'application/json',
    Accept: 'application/json, text/event-stream',
    Authorization: `Bearer ${process.env[tokenEnv]}`,
  };
  let id = 0;
  const rpc = async (method, params) => {
    const res = await fetch(url, {
      method: 'POST',
      headers,
      body: JSON.stringify({ jsonrpc: '2.0', id: ++id, method, params }),
    });
    const body = await res.json();
    if (body.error) {
      throw new Error(`${method}: ${body.error.message}`);
    }
    return body.result;
  };
  await rpc('initialize', {
    protocolVersion: '2025-06-18',
    capabilities: {},
    clientInfo: { name: 'fake-codex', version: '1' },
  });
  const { tools } = await rpc('tools/list', {});
  const tool = tools.find((t) => t.name.endsWith('__create_note'));
  if (tool === undefined) {
    return 'ferramenta create_note ausente';
  }
  const result = await rpc('tools/call', { name: tool.name, arguments: { text: 'nota do teste E2E' } });
  return result.content.map((c) => c.text).join(' ');
}
