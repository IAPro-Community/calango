# ADR 0016 — Provider chain per agent with fallback and named accounts

Status: accepted · 2026-11-23 · owner decision (Q22 routing question) · extends ADR 0013 and ADR 0014.

## Context

ADR 0013 made each harness a `gateway.Provider` and the daemon wires one provider for every run (`conversations.Config.Provider`). The owner asked for:

- each agent picks its own provider;
- several accounts per provider (for example two Codex logins, two Claude logins, an Antigravity account, an API key, a local model);
- fallback to the next provider when one fails, so an agent does not stop.

`Agent.Harness` holds one provider name today. `Agent.Model` is one model string. Neither can express an ordered chain.

## Decision

1. **Account.** A provider account is a named, owner-scoped row: `kind` (`codex`, `anthropic`, later `claude-code`, `opencode`, `antigravity`, `openai`, `local`), `label`, and a non-secret `settings` object. Secrets never go into the row:
   - API keys stay in the vault (`anthropic` uses `anthropic-api-key`; more accounts use `<kind>-<label>-api-key`).
   - CLI logins live in an isolated home per account, `<data-dir>/accounts/<kind>/<label>`, passed as `CODEX_HOME` (or the equivalent for each CLI). Sign-in happens with the CLI outside Calango; Calango never copies `auth.json` (ADR 0013, ADR 0014).
2. **Chain.** An agent has an ordered list `providers`: `{account_id, model}`. The first entry is tried first. The list has at least one entry and at most 8.
3. **Fallback rule.** The next entry runs only when the attempt fails before any output chunk was streamed to the run. Failures that qualify: CLI not installed, signed out, HTTP 429, HTTP 5xx, transport error, stream error before the first chunk. Not qualifying: budget exceeded (per agent, same for every entry), cancellation by the owner, validation errors. A failure after output started ends the run as `failed` with the error, so the owner never gets two half-answers spliced together.
4. **Accounting.** Every attempt is one gateway call. The usage record carries that attempt's account and model. The run records each attempt as a step (`provider_attempt`, status, error code) so the fallback path is visible.
5. **Migration.** Migration `0010` adds `provider_accounts` and `agent_providers` (position, agent_id, account_id, model). The columns `agents.harness` and `agents.model` are dropped in the same migration after copying them into the chain; no shim keeps them.
6. **Scope of V1.** Accounts and chains are data and routing. Only `codex` and `anthropic` have adapters in this ADR's first step. Other kinds are rejected by validation until their adapter exists (F1-T05 and the open harnesses).

## Consequences

- Agents API changes: `harness` and `model` are replaced by `providers`. OpenAPI and clients follow in the same change.
- Codex accounts need one sign-in each, done by the owner with `CODEX_HOME=<account home> codex login`. Calango reports `harness_signed_out` per account.
- Fallback can spend more tokens (a failed attempt may still be charged the reserved estimate, as the gateway already does). The budget check runs per attempt.
- Per-account credentials are isolated, so removing an account deletes its home and vault item only after confirmation.

## Alternatives rejected

- One provider per run chosen by the owner: no fallback, no unattended operation.
- Accounts stored inside the agent row as JSON: hides secrets' location and makes account reuse across agents impossible.
- Fallback after partial output: duplicates or contradicts text the owner already read.

## Amendment (implementation, 2026-11-23)

- **No settings object.** An account row holds `id`, `owner_id`, `kind`, `label` and timestamps. Options specific to a kind come with its adapter; clients do not send settings.
- **Default account.** Migration 0010 creates one `default` account per owner and harness that had agents. Codex `default` has an empty home, so it shares `~/.codex` (ADR 0013) and existing logins keep working. Anthropic `default` uses the vault item `anthropic-api-key`.
- **Other accounts.** Codex home `<data-dir>/accounts/codex/<label>` (mode 0700). Anthropic vault item `anthropic-<label>-api-key`. Labels are 1–32 characters.
- **Fallback classification.** Anthropic wraps `gateway.ErrUnavailable` around transport errors, HTTP 429 and 5xx, stream errors and streams cut before `message_stop`. Codex wraps `harness.ErrNotInstalled` and `harness.ErrSignedOut`. Any other error ends the attempt without fallback, including HTTP 401 (bad key).
- **Empty chain.** A run of an agent with no `providers` fails at once with `no_provider`. Migration 0010 gives no chain to agents that had no model, so they must be saved again with `providers` before they run.
- **Delete.** `DELETE /v1/provider-accounts/{id}` answers 409 while a chain uses the account. It removes the account row and, for Codex, its home (never `default`). Anthropic vault items are not touched; the owner removes them with `DELETE /v1/vault/{name}`.
- **API.** `providers` (1–8 entries, required) replaces `harness` and `model` on agents; responses carry `account` (the gateway key `<kind>/<label>`). Routes: `GET/POST /v1/provider-accounts`, `DELETE /v1/provider-accounts/{id}`. Only the `codex` and `anthropic` kinds are accepted. The `setup` field says how to sign in (`CODEX_HOME=<home> codex login`) or where to put the key.


## Amendment (SDK integration, proposed)

See ADR 0020. Retry requires a safe boundary, not just absence of output. Gateway obtains a usage ID before dispatch and returns ErrLedger for accounting failure. The adapter explicitly declares whether external side effects are possible; undeclared means possible. Native CLI preflight failures remain retryable, ambiguous post-dispatch failures do not. No automatic fallback outside the configured chain.
