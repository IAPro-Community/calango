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
| `CALANGO_PRICE_TABLE_FILE` | unset | Absolute path to a versioned JSON tariff file in USD micros per million tokens. Invalid configured files stop daemon startup; `print-token` ignores tariff configuration. Unlisted account/model combinations remain unknown. See below. |

`internal/config.Load` is the only point that reads `CALANGO_*` variables. New variables must come here with a default, validation, usage and documentation. Secrets must not be placed in environment variables when the system vault is supported. The one exception is the VPS vault master key (`CALANGO_VAULT_KEY`, `CALANGO_VAULT_KEY_FILE`, and the `_VERSION`, `_PREVIOUS` and `_PREVIOUS_FILE` variables during rotation), because a VPS has no OS vault (ADR 0012).

## Invariants

- The default listens only on loopback.
- Invalid configuration ends the process with an error before the scheduler starts.
- A public address is opt-in and must be protected by the topology documented in `DEV/DEPLOYMENT.md`.
- Errors do not print credential values. `Config` can hold the vault master key; never print or log a `Config`.

## Verification

- Unit tests cover the default, an explicit address, invalid formats/ports, and the vault key settings (mutually exclusive sources).
- `go test ./...` and `go vet ./...` run in CI.

## Price table (#60)

The optional file contains no credentials. Synthetic example, **not a real price**:

```json
{"version":"example-not-a-real-price","currency":"USD","models":{"anthropic/default/example-model":{"input_per_million":1000000,"output_per_million":2000000}}}
```

The key uses the exact provider/account identifier followed by the model sent to the gateway. No tariff fallback across accounts or providers. Values are integers in USD micros per million tokens. Both are required; zero is valid only when explicitly configured. Without a file or matching entry, cost remains unknown. Changes require restart and a new table version; historical prices are not recomputed. Configured price multiplied by measured tokens does not prove actual provider billing, especially for subscriptions, caches and discounts; do not configure a tariff without validating the account's billing mode. Real-cost acceptance remains open in #60.

The tariff must be a regular file (at most 1 MiB); duplicate JSON keys are rejected. The operator must keep the configured path under trusted control.
