#!/usr/bin/env node
// Minimal stdio MCP server for the desktop E2E tests: one write tool, create_note,
// so the MCP gateway asks for an approval before it runs. Messages are newline
// delimited JSON-RPC, as the MCP stdio transport specifies.
import { createInterface } from 'node:readline';

const tools = [
  {
    name: 'create_note',
    description: 'Save a note (E2E fixture).',
    inputSchema: { type: 'object', properties: { text: { type: 'string' } }, required: ['text'] },
    annotations: { readOnlyHint: false },
  },
];

function reply(id, result) {
  process.stdout.write(`${JSON.stringify({ jsonrpc: '2.0', id, result })}\n`);
}

for await (const line of createInterface({ input: process.stdin })) {
  if (line.trim() === '') {
    continue;
  }
  const msg = JSON.parse(line);
  if (msg.id === undefined) {
    continue; // notification
  }
  switch (msg.method) {
    case 'initialize':
      reply(msg.id, {
        protocolVersion: msg.params.protocolVersion,
        capabilities: { tools: {} },
        serverInfo: { name: 'e2e-notes', version: '1' },
      });
      break;
    case 'tools/list':
      reply(msg.id, { tools });
      break;
    case 'tools/call':
      reply(msg.id, { content: [{ type: 'text', text: `nota salva: ${msg.params.arguments?.text ?? ''}` }] });
      break;
    case 'ping':
      reply(msg.id, {});
      break;
    default:
      process.stdout.write(
        `${JSON.stringify({ jsonrpc: '2.0', id: msg.id, error: { code: -32601, message: 'method not found' } })}\n`,
      );
  }
}
