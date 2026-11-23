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
