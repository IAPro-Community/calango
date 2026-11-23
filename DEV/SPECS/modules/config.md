# Process configuration

## Goal

Load environment configuration in a single package and reject invalid values before starting the server or background tasks.

## Current contract

| Variable | Default | Validation |
|---|---|---|
| `CALANGO_ADDR` | `127.0.0.1:7317` | `host:port` address; host required; numeric port from 1 to 65535. Use `0.0.0.0` only when the node must accept network connections, such as in the container behind a TLS proxy. |

`internal/config.Load` is the only point that reads `CALANGO_*` variables. New variables must come here with a default, validation, usage and documentation. Secrets must not be placed in environment variables when the system vault is supported.

## Invariants

- The default listens only on loopback.
- Invalid configuration ends the process with an error before the scheduler starts.
- A public address is opt-in and must be protected by the topology documented in `DEV/DEPLOYMENT.md`.
- Errors do not print credential values; the current configuration contains no secrets.

## Verification

- Unit tests cover the default, an explicit address, and invalid formats/ports.
- `go test ./...` and `go vet ./...` run in CI.
