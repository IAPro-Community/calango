# ADR 0012 — Vault: sealed values in SQLite, master key from the OS keyring or the deployment

Status: accepted · 2026-11-23 · ports the design of `core/vault.py` from [ADR 0011](0011-reuso-agenticos.md)

## Context
[SPECS/modules/vault.md](../SPECS/modules/vault.md) requires secrets to live in the operating system vault on the desktop (Windows Credential Manager, macOS Keychain, Linux Secret Service) and, on the VPS, in an encrypted file whose key comes from an environment variable or a Docker secret, outside the data volume. Linux without an unlocked keyring must refuse to save, never fall back to plain text.

Storing every secret directly in the OS vault cannot work: Linux Secret Service has no list operation for names, and the VPS has no OS vault. The upstream design (`backend/app/core/vault.py`) solves a similar problem with envelope encryption: a random data key per secret, wrapped by a key derived from a master key and the owner scope, with the master key version recorded on each row.

## Decision
1. **Values** are stored in SQLite (`vault_items`, migration 0009) as envelopes: a random 256-bit data key seals the value with AES-256-GCM; the data key is wrapped with a key derived by HKDF-SHA256 from the master key, the owner scope and the master key version. The scope is also bound as additional data on both ciphertexts.
2. **Master key** comes from one of two sources, chosen by the deployment:
   - desktop: the OS keyring (`github.com/zalando/go-keyring`), created on first use under service `calango`, account `vault-master-v<N>`;
   - VPS: `CALANGO_VAULT_KEY` or `CALANGO_VAULT_KEY_FILE` (Docker secret), base64 of 32 bytes, kept outside the data volume.
   A missing master key refuses every write. There is no plain-text fallback.
3. **Rotation** adds a new master key version and re-wraps each data key (`Rewrap`); payloads are not re-encrypted. Old versions stay configured until no row names them.
4. **Reveal** returns the plain value only to the in-process caller that uses it. The API returns name and dates only, never the value. Logs and steps never carry the value.
5. Envelope format starts at version 1 in Calango. No compatibility with the upstream envelope: Calango has no stored data to migrate.
6. **API token and startup.** The local API token is the vault item `api-token`, created on first start as 256 random bits. A legacy `DataDir/api-token` file (mode 0600) is imported into the vault and deleted; if the vault already holds a different token, the start fails and neither value is touched. The daemon does not start without a master key. `internal/config` reads `CALANGO_VAULT_KEY` and `CALANGO_VAULT_KEY_FILE` (mutually exclusive) and passes them to `internal/vault`; no other package reads them.

## Consequences
- The vault works the same way on desktop and VPS; only the master-key source differs.
- Desktop loss of the OS keyring entry makes every stored secret unreadable. The owner must be told this in the UI when the key is created (pending in F1-T10 UI work).
- Linux desktops without Secret Service cannot store secrets until the keyring is unlocked. This is intended.
- A database copy without the master key reveals no value.
- The daemon refuses to start without a master key. Linux runs without Secret Service need `CALANGO_VAULT_KEY_FILE` before the first start, or an unlocked keyring. This replaces the earlier file-based token, which started without any key; the owner must acknowledge it.
- Dependencies added: `github.com/zalando/go-keyring` v0.2.8 (MIT), with `github.com/godbus/dbus/v5` (BSD-2-Clause) on Linux and `github.com/danieljoos/wincred` (MIT) on Windows. Listed in `NOTICE` and the Go module graph.

## Alternatives rejected
- **Secrets only in the OS vault.** No listing on Linux, and no vault on the VPS.
- **One global key in the config file.** Gives no binding between owners; the upstream project removed this design for that reason.
- **Porting upstream Fernet.** Fernet uses AES-128-CBC with HMAC and a fixed format; GCM with explicit nonces is the boring choice in Go's standard library.
- **HKDF from `golang.org/x/crypto/hkdf`.** Needs a new module; the one-block HKDF used here is about 15 lines over `crypto/hmac`, with the RFC 5869 structure.
