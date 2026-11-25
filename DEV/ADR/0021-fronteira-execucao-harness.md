# ADR 0021 — Harness execution boundary

Status: proposed, human review required · Refs #113, #53, #54, #57, #117, #128.

## Current guarantees

| Capability | Direct Anthropic API | Codex CLI |
|---|---|---|
| Calango MCP tools | Does not execute tools | Per-turn gateway; grants and write approval |
| Native commands | Does not execute | CLI and its own sandbox; bypass the gateway |
| Filesystem | No local tools | workspace-write; reads and isolation depend on CLI/OS |
| Network | HTTP to provider | CLI/OS policy; not controlled by MCP |
| Budget | Reservation + max_tokens sent | Reservation and ledger; hard output limit not proven |
| Cancellation | Request context | Cancels main process; process-tree cleanup not proven |
| Credentials | Account key through vault | Local login and CODEX_HOME; environment explicitly filtered |

ReadOnlyHint is an MCP server declaration, not proof of absence of side effects. Connectors must be trusted and authorized in advance. External actions with ambiguous outcomes must not be automatically replayed.

## Proposed policy

- CLI environment allowlist; no inherited Calango key, other-turn token, API key or Git token. The current MCP token is injected exclusively for the turn. Login uses the configured account. Do not accept implicit environment credentials as account identity.
- PR #135 must preserve this policy when migrating the command factory to the SDK. Do not create a second parser/runtime.
- Do not promise authorization of all native commands. Until isolation is proven, unrestricted autonomous execution is not accepted. Define a read-only profile and an explicitly consented workspace profile; validate flags and limits against real provider versions before enabling them.
- Do not promise descendant cleanup without native tests. The SDK is the common runtime authority; the Calango host retains permission, budget and audit decisions.
- Do not promise confidentiality against malicious processes running as the same user; filtering the environment does not prevent reads of files, sockets, credentials or other capabilities available to that user.

## Remaining gates

Fixtures: environment, secrets, account isolation, revocation, cancellation before/during output, missing receipts and fallback policy. Windows acceptance for processes and descendants, and real Codex/Anthropic with an explicit budget. Linux/macOS before announcing support. Profile approval, UI communication of limitations and public documentation. Do not close #113/#41 solely because CI is green.
