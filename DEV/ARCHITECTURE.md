# Architecture — Calango

## Deployment modes

- **Person's VPS (Docker)**: calango runs on the VPS; desktop, iOS and Android connect over TLS. See [DEPLOYMENT.md](DEPLOYMENT.md).
- **Local computer**: same binary, local address.
- **Hosted by us** (paid): same API, with no server administration by the user.

Clients never run agents. They only show state and send commands.

## Principle

One core, in Go. Local, cloud and clients are shells around it. No agent logic on the client.

## Layers

```
clients (Electron desktop, iOS, Android, web)
        │  HTTP + WebSocket
calango  (internal/api)
        │
core      (internal/core: agents, channels, routines, budget, approvals)
        │
storage (SQLite, Store interface)   ·   adapters (models, browser, MCP)
```

## Packages

- `cmd/calango`: daemon entry point. Starts the server and the scheduler; shuts down on signal.
- `internal/core`: pure domain, no network I/O.
  - `agent.go`: agent with token budget (`Charge` refuses when it overflows).
  - `scheduler.go`: routines by interval, with an injectable clock.
- `internal/api`: HTTP routes.

## Rules

- Model calls go through a single gateway that checks the budget before and records the cost after.
- Secrets never go back to the client.
- Sensitive write actions (send, delete, publish, pay) require explicit owner approval.
- Every entity has an owner (`owner`) to support multiple accounts in the cloud later.
