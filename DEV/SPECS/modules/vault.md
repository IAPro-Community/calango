# Module: vault

Phase 1 · Package `internal/vault` · Table `vault_items` (reference only)

## Goal
Store secrets (model keys, connector tokens, site passwords) in the operating system's vault.

## Rules
- Backends: Windows Credential Manager (DPAPI), macOS Keychain, Linux Secret Service. On the VPS (no OS vault): an encrypted file with a key from an environment variable or Docker secret, kept outside the data volume.
- Linux without an unlocked keyring: refuse to save and explain how to unlock it. Never fall back to plain text.
- The API returns only the name and date; the value never comes back.
- The agent uses a secret by reference (`vault://name`); the value is resolved at the time of the call and masked in logs and steps.
- An agent can ask the person for a secret (`request_secret`): a card where the person fills it in; the agent is notified when the value is available.

## Acceptance
- No secret appears in `run_steps`, logs or API responses (canary test).
