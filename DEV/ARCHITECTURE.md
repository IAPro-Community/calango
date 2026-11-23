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

- `cmd/calango`: daemon entry point. Starts the HTTP server; shuts down on signal. The routines loop starts once an executor exists (F1-T09 pending).
- `internal/core`: pure domain, no network I/O.
  - `agent.go`: agent with token budget (`Charge` refuses when it overflows).
  - `routine.go`, `cron.go`: routine definition, validation, 5-field cron, retry backoff, quiet window and occurrence ID (pure).
- `internal/routines`: routine service on the store: occurrences, retries, quiet windows, coalescing of missed runs, notifications.
- `internal/api`: HTTP routes.

## Rules

- Model calls go through a single gateway that checks the budget before and records the cost after.
- Secrets never go back to the client.
- Sensitive write actions (send, delete, publish, pay) require explicit owner approval.
- Every entity has an owner (`owner`) to support multiple accounts in the cloud later.
