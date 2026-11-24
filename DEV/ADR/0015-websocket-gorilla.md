# ADR 0015 — WebSocket `/v1/events` with gorilla/websocket

Status: accepted · 2026-11-23 · implements the WebSocket part of F1-T13 (Q23).

## Context

The API must push conversation changes (messages, run start, steps, run end) to connected clients. The queue (Q23) forbids an SSE substitute. The project targets Go 1.22 (`go.mod` line 3). The standard library has no WebSocket server.

Candidates checked against the module proxy (`proxy.golang.org`: `.mod`, `.info`, LICENSE):

| Module | Version | Licence | `go` directive | Result |
|---|---|---|---|---|
| `github.com/gorilla/websocket` | v1.5.3 | BSD-2-Clause | `go 1.12` | chosen |
| `github.com/coder/websocket` | v1.8.14 | see proxy LICENSE | `go 1.23` | rejected: needs a newer toolchain than the project declares |

## Decision

1. Use `github.com/gorilla/websocket` v1.5.3. Licence recorded in `NOTICE`.
2. Authentication is the `Authorization: Bearer` header, checked by the existing `need(core.PermRead)` middleware before the upgrade. No token in the URL (URLs reach logs), no cookies. `CheckOrigin` accepts any origin because the header, not the origin, is the credential.
3. Fan-out is in-process (`internal/events.Hub`). Each subscriber has a 64-event buffer. A full buffer closes the subscription; the handler then sends close 1013 and the client reloads state over REST.
4. No replay and no event ids. Reconnection = REST reload of conversations and runs.
5. Heartbeat: server ping every 25 s; a client silent for 60 s is dropped.
6. Subscription happens before the upgrade, so a client that has received 101 cannot miss an event published after it.

## Consequences

- One new direct dependency. The handler uses only `Upgrader`, `Conn` read/write, ping/pong and close frames, so replacing the library later stays a local change.
- Slow clients lose their connection rather than stalling publishers. Publishing never blocks the conversation service.
- Desktop must connect from the Electron main process and forward to the renderer over IPC, so the token never enters renderer code. Not built yet.
- Mobile and other clients need a reconnect loop that reloads REST state.

## Alternatives rejected

- `coder/websocket`: requires Go 1.23; raising the project's Go version is a separate decision.
- SSE: excluded by the queue.
- Polling REST: higher latency and load; not what Q23 asks for.
