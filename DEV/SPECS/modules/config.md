# Process configuration

## Goal

Load environment configuration in a single package and reject invalid values before starting the server or background tasks.

## Current contract

| Variable | Default | Validation |
|---|---|---|
| `CALANGO_ADDR` | `127.0.0.1:7317` | `host:port` address; host required; numeric port from 1 to 65535. Use `0.0.0.0` only when the node must accept network connections, such as in the container behind a TLS proxy. |
| `CALANGO_VAULT_KEY` | unset | Vault master key: base64 of 32 bytes, for hosts without an OS keyring. Mutually exclusive with `CALANGO_VAULT_KEY_FILE`. Secret: never logged; errors do not echo it. |
| `CALANGO_VAULT_KEY_FILE` | unset | Path to a file with the same base64 key (for example a Docker secret kept outside the data volume). Mutually exclusive with `CALANGO_VAULT_KEY`. |

`internal/config.Load` is the only point that reads `CALANGO_*` variables. New variables must come here with a default, validation, usage and documentation. Secrets must not be placed in environment variables when the system vault is supported. The one exception is the VPS vault master key (`CALANGO_VAULT_KEY`, `CALANGO_VAULT_KEY_FILE`), because a VPS has no OS vault (ADR 0012).

## Invariants

- The default listens only on loopback.
- Invalid configuration ends the process with an error before the scheduler starts.
- A public address is opt-in and must be protected by the topology documented in `DEV/DEPLOYMENT.md`.
- Errors do not print credential values. `Config` can hold the vault master key; never print or log a `Config`.

## Verification

- Unit tests cover the default, an explicit address, invalid formats/ports, and the vault key settings (mutually exclusive sources).
- `go test ./...` and `go vet ./...` run in CI.
