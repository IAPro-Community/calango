# Process configuration

## Goal

Load environment configuration in a single package and reject invalid values before starting the server or background tasks.

## Current contract

| Variable | Default | Validation |
|---|---|---|
| `CALANGO_ADDR` | `127.0.0.1:7317` | `host:port` address; host required; numeric port from 1 to 65535. Use `0.0.0.0` only when the node must accept network connections, such as in the container behind a TLS proxy. |
| `CALANGO_VAULT_KEY` | unset | Vault master key: base64 of 32 bytes, for hosts without an OS keyring. Mutually exclusive with `CALANGO_VAULT_KEY_FILE`. Secret: never logged; errors do not echo it. |
| `CALANGO_VAULT_KEY_FILE` | unset | Path to a file with the same base64 key (for example a Docker secret kept outside the data volume). Mutually exclusive with `CALANGO_VAULT_KEY`. |
| `CALANGO_VAULT_KEY_VERSION` | `1` | Version of the current explicit key (`CALANGO_VAULT_KEY` or `_FILE`). Integer ≥ 1. Above 1 needs an explicit current key; the OS keyring stays at version 1. See the ADR 0012 amendment. |
| `CALANGO_VAULT_KEY_PREVIOUS` | unset | Old master key during rotation: base64 of 32 bytes. Needs an explicit current key and version ≥ 2. Mutually exclusive with `CALANGO_VAULT_KEY_PREVIOUS_FILE`. Remove after the rewrap is logged. Secret: never logged; errors do not echo it. |
| `CALANGO_VAULT_KEY_PREVIOUS_FILE` | unset | Path to a file with the same old key. Mutually exclusive with `CALANGO_VAULT_KEY_PREVIOUS`. |
| `CALANGO_TRUSTED_PROXIES` | unset | Comma-separated IP addresses or CIDR ranges of reverse proxies. When the TCP peer is listed, the `/v1/pair` limit uses the right-most `X-Forwarded-For` entry that is not a listed proxy. Empty trusts none. Malformed entries fail startup; errors name the variable only. |
| `CALANGO_LICENSE_FILE` | `<CALANGO_DATA_DIR>/calango.license` | Absolute path to the signed license token. Missing or invalid license means the Free plan; it never stops the node. See `licensing-billing.md`. |
| `CALANGO_LICENSE_PUBLIC_KEY` | unset | Base64 Ed25519 public key (32 bytes) that verifies licenses in builds without an embedded key; ignored when the build embeds one. Public, not a secret. A malformed value fails startup. |

`internal/config.Load` is the only point that reads `CALANGO_*` variables. New variables must come here with a default, validation, usage and documentation. Secrets must not be placed in environment variables when the system vault is supported. The one exception is the VPS vault master key (`CALANGO_VAULT_KEY`, `CALANGO_VAULT_KEY_FILE`, and the `_VERSION`, `_PREVIOUS` and `_PREVIOUS_FILE` variables during rotation), because a VPS has no OS vault (ADR 0012).

## Invariants

- The default listens only on loopback.
- Invalid configuration ends the process with an error before the scheduler starts.
- A public address is opt-in and must be protected by the topology documented in `DEV/DEPLOYMENT.md`.
- Errors do not print credential values. `Config` can hold the vault master key; never print or log a `Config`.

## Verification

- Unit tests cover the default, an explicit address, invalid formats/ports, the vault key settings (mutually exclusive sources) and the license file/public key settings.
- `go test ./...` and `go vet ./...` run in CI.
