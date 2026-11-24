# ADR 0018 — MCP gateway per turn: token in the environment, approvals for writes

Status: accepted · 2026-11-23 · implements F1-T12 (Q20) · builds on ADR 0011 (P3), ADR 0013 (Codex sandbox), ADR 0016 (provider chain).

## Context

Codex runs with `sandbox: workspace-write` (ADR 0013). Until MCP routes its tools through Calango, Codex can write files and run commands without approval. F1-T12 must put Calango's tool list and approvals in the middle of that path.

The module spec (`DEV/SPECS/modules/tools-mcp.md`) describes a **secret URL** per turn that lists only that agent's tools. Passing the secret in the URL puts it in the Codex argv and in any process listing and log that records the command line.

The MCP client and server come from the official Go SDK, `github.com/modelcontextprotocol/go-sdk` v1.3.0 (MIT). It needs Go 1.23 (owner decision, 2026-11-23; see `DEV/HANDOFF.md`).

## Decision

1. **Token in the environment, not the URL.** Each run that has at least one grant gets a 32-byte random token, hex encoded. The gateway stores only its SHA-256 digest, in memory, for the turn. The token reaches Codex as the environment variable `CALANGO_MCP_TOKEN`, referenced from the config as `mcp_servers.calango.bearer_token_env_var`. The URL (`http://127.0.0.1:<port>/mcp`) is fixed per daemon and carries no secret. The token never appears in argv. This deviates from the spec's "secret URL" and is the reason for this ADR.
2. **Per-turn scope.** `mcpgw.Gateway.Open` returns no endpoint when the agent has no grants. Otherwise the token maps to (owner, agent, run). The turn ends when `release` is called (`defer` in the conversations service). A token from a finished turn is rejected.
3. **Grants.** One row per (agent, connector) in `agent_tools`, access `read` or `write`. No row means none.
   - `read` exposes only tools whose annotations set `readOnlyHint: true`.
   - `write` exposes every tool. Calls to read-hinted tools run directly. Calls to the others go through approvals.
4. **Approvals for writes.** The gateway calls `approvals.Request` with kind `core.ActionConnectorWrite` and target `<connector>/<remote tool>`. The request carries the run ID, so the ownership check from Q15 applies. The gateway polls the approval for a bounded window (`DefaultWaitWindow`, 2 min). Execution is done by `Execute`, registered with `approvals.SetExecutor`, from the stored snapshot. Results are kept in memory for one hour (`resultTTL`). The approval outcome is recorded by the approvals service (`completed` or `failed` with reason).
5. **Stdio connectors get a minimal environment.** The child process receives `PATH` and `HOME` only. Connector names are `^[a-z0-9][a-z0-9-]{0,23}$`; exposed tool names are `<connector>__<remote tool>` and remote names must match `^[A-Za-z0-9_-]+$`.
6. **Stateless transport.** `Stateless: true, JSONResponse: true`. No MCP session survives a request. The remote tool list is queried per request; no cache in V1. A connector that fails to list is skipped, with a warning in the log.
7. **Mounting.** `/mcp` is mounted on the root mux outside the API's bearer middleware. The gateway checks its own token. The API keeps its `/v1` surface unchanged.

## Consequences

- Codex receives a tool list that holds only what the agent is granted. A hidden or ungranted tool gets the SDK's "unknown tool" error, not a custom refusal. A custom refusal with "what to do instead" needs a custom handler; this is a gap.
- A write call holds the Codex turn open while it waits for approval, up to 2 min. The run stays `running` during that wait. This is the known limit of Q15 (`waiting_approval` not yet wired).
- Results of approved writes are lost on restart (memory only, one hour). An approved action that crashed before its result is not reconciled (Q15).
- Connectors have no credentials in V1. Anything that needs an API key or OAuth is out of scope until a vault-backed credential design is approved.
- Codex's config key names (`mcp_servers.<name>.url`, `bearer_token_env_var`) are unverified against a live Codex 0.171 run. Owner authorisation is needed for a real model call, so this stays open.
- With `sandbox: workspace-write`, Codex may block loopback access to the gateway. This is unverified and must be checked in the first live run.
- Anthropic direct and other providers ignore the `Tools` field in V1: their agents get no Calango tools. Stated in `DEV/SPECS/modules/tools-mcp.md`.
- Test coverage: `internal/mcpgw` (token scope, approval gating, execution from snapshot), `internal/tools` (catalog, grants, unavailable connectors, stdio environment), `internal/harness` (token only in child env), `internal/api` (connector and grant routes).

## Not decided

- Credentials for remote connectors (vault-backed, per connector).
- Restart durability for approved results and reconciliation of approved actions without a receipt (Q15).
- A custom refusal message for hidden tools.

## Amendment (2026-11-23): Q15 wired, built-in memory_save

- Q15 is done. A write waiting for approval moves the run to `waiting_approval` (`mcpgw.callWithApproval`). `approvals.Reconcile` at startup closes approved actions with no receipt as failed. The Consequences above that say Q15 is not wired no longer hold.
- Decision 2 is replaced. `Open` returns an endpoint when the agent has grants or the gateway has a memory service, which the daemon always wires. An agent with no grants gets only `memory_save`.
- `memory_save` is a built-in tool with no connector prefix. It saves an agent-scoped memory of the turn's agent; the scope and agent come from the turn, never from the arguments. It needs no approval: the person sees, edits and deletes every memory (`DEV/SPECS/modules/memory.md`).
- Other built-ins (`next_steps`, `ask_person`, `routine_finish`, `task_update`) are still not built.
- The live Codex check remains unverified.
