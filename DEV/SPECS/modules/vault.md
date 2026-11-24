# Module: vault

Phase 1 · Package `internal/vault` · Table `vault_items` (reference only)

Source: ADR 0011 (P1) and ADR 0012 (storage design). Ported from `backend/app/core/vault.py` in `vstorm-co/agenticos` at a4b0b3c.

## Goal
Store secrets (model keys, connector tokens, site passwords) in the operating system's vault.

## Rules
- Storage: each value is an envelope (AES-256-GCM, wrapped data key bound to owner and key version) in table `vault_items`. The master key comes from the OS keyring on the desktop (`calango` / `vault-master-v<N>`) or from `CALANGO_VAULT_KEY_FILE` / `CALANGO_VAULT_KEY` on the VPS (Docker secret, outside the data volume). Backends: Windows Credential Manager, macOS Keychain, Linux Secret Service via `zalando/go-keyring`.
- Linux without an unlocked keyring, or no master key configured: refuse to save and explain how to unlock it. Never fall back to plain text.
- The API returns only the name and date; the value never comes back.
- HTTP (`/v1/vault`): list names and dates; set and delete on `/v1/vault/{name}`. Full scope only. No route returns a value. The name `api-token` is reserved: set and delete return 409. The local token changes only through `POST /v1/auth/rotate-token`, which writes the vault item directly.
- Set, reveal and delete append an audit event (`vault.set`, `vault.reveal`, `vault.delete`; actor and name only, never the value) before they act. If the audit write fails the action is refused: a secret is never changed or revealed unrecorded.
- The agent uses a secret by reference (`vault://name`); the value is resolved at the time of the call and masked in logs and steps.
- Master-key rotation at startup (`CALANGO_VAULT_KEY_VERSION`, `CALANGO_VAULT_KEY_PREVIOUS[_FILE]`): each secret sealed under another key version is re-wrapped to the current one before the daemon serves. Each re-wrap is audited as `vault.rewrap` (name only). Procedure and refusal rules: ADR 0012 amendment.
- An agent can ask the person for a secret (`request_secret`): a card where the person fills it in; the agent is notified when the value is available.

## Acceptance
- No secret appears in `run_steps`, logs or API responses (canary test).
