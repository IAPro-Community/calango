# Module: tools-mcp

Phase 1 · Packages `internal/tools`, `internal/mcpgw` · Tables `connectors`, `agent_tools`

Source: ADR 0011 (P3). Ported from `backend/app/agents/mcp.py`, `services/mcp_connection.py` and `services/mcp_audit.py` in `vstorm-co/agenticos` at a4b0b3c; the MCP client comes from the official Go SDK where it covers the protocol.

## Goal
Agent tools via MCP, scoped per agent, with approval for anything that writes.

## Rules
- Connect an MCP server once (stdio or HTTP); each agent receives a subset of tools with `none`, `read` or `write` access.
- **Local MCP gateway**: on each turn, a secret URL that lists only that agent's tools. It is the only tool source for the harness.
- A `write` tool call goes through `approvals`.
- The MCP server's instructions (`instructions`) are passed on to the agent when the connection allows it.
- Calango's own tools also go through the gateway: `memory_save`, `next_steps`, `ask_person`, `routine_finish`, `task_update`.
- The node also exposes an **MCP server** so external Claude Code/Codex can talk to Calango (phase 4).
- A refusal by rule returns **what to do instead**, so the agent does not repeat the same call.

## Acceptance
- An agent without access to a tool does not see it in the list.
- A `write` tool without approval does not run.

## Status (F1-T12 / Q20, 2026-11-23)
- Built: `connectors` and `agent_tools` (migration 0013); `internal/tools` (register, list, delete, grants, catalog, remote calls; stdio with `PATH` and `HOME` only); `internal/mcpgw` (per-turn token, gateway at `/mcp`, read and write scopes, writes through approvals); built-in `memory_save` (agent-scoped, no approval; ADR 0018 amendment); Codex harness wiring (`internal/harness`); conversations open and release one endpoint per run; HTTP routes `/v1/connectors*` and `/v1/agents/{id}/tools*` (OpenAPI updated).
- Decision: the per-turn token goes in the environment variable `CALANGO_MCP_TOKEN`, not in the URL. See `DEV/ADR/0018-mcp-gateway-por-turno.md`.
- Not built: connector credentials; Calango's own tools other than `memory_save` (`next_steps`, `ask_person`, `routine_finish`, `task_update`); the MCP server for external clients (phase 4); the MCP server's `instructions`; a custom refusal for hidden tools; a cached tool list; restart durability for approved results.
- Anthropic direct and other non-Codex providers ignore the gateway in V1.
- Live check still open: Codex 0.171 with `sandbox: workspace-write` reaching the loopback gateway and the `mcp_servers` keys. Needs owner authorisation for a model call.
