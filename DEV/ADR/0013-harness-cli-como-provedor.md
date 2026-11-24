# ADR 0013 — Agent CLI harness runs as a gateway provider

Status: accepted · 2026-11-23 · refines the `Harness` interface in [SPECS/modules/harness.md](../SPECS/modules/harness.md)

## Context
[SPECS/modules/harness.md](../SPECS/modules/harness.md) defines a `Harness` interface with `Run(ctx, turn, sink)` and a `Turn` that carries memory, history and tool list. The first V1 adapter is the Codex CLI (owner decision D3). Every model call must pass through `internal/gateway` (budget checked before, cost recorded after; [ADR 0007](0007-gateway-de-modelo.md)).

Building `Run(ctx, turn, sink)` now would duplicate the gateway's budget and usage path and would require the conversation service to know about CLIs. The conversation service already calls `gateway.Call` with a `Provider` name and a prompt, so a CLI fits that contract as one more provider.

## Decision
1. An agent CLI is a `gateway.Provider`. `internal/harness.Codex` implements `Name()` and `Complete(ctx, Request, stream)`. The gateway therefore enforces budget, records usage and returns errors the same way for CLIs and direct APIs.
2. The `Run(ctx, turn, sink)` interface stays the target for F1-T07/T11 (steps, memory, tool list, MCP gateway). It is not built in this slice.
3. Each run is a one-shot `codex exec` with `--ephemeral --ignore-user-config --ignore-rules --skip-git-repo-check`, `--sandbox workspace-write`, `-C <working dir>`, `-o <temp file>`, prompt on stdin. Stdout and stderr are discarded: they are not echoed into errors, because they may contain the prompt.
4. **Isolation** shares the person's `~/.codex` (auth stays there; no copy of the token) and turns off the user's config and rules for the run. This is weaker than a temporary `CODEX_HOME`, which is accepted for V1 by owner choice. Whether `~/.codex/AGENTS.md` still loads under `--ignore-user-config` is unverified.
5. **Sign-in** is checked with `codex login status` before each run. A signed-out CLI returns `ErrSignedOut`; the conversation service maps it to `failed/harness_signed_out`.
6. **Tokens** are estimated (about 4 characters per token) and marked `Estimated`, because the `--json` event schema is not verified yet. The reply is one stream chunk, not token streaming.
7. **Cancellation.** A cancelled context kills the CLI. `cmd.WaitDelay` (2 s) closes the pipes if a child process keeps them open. Killing the whole process group is not done (needs platform-specific code); this is a known limit.

## Consequences
- The conversation service needs no CLI-specific code; it gains one error code, `harness_signed_out`.
- `sandbox: workspace-write` lets Codex run commands and write files without Calango approval. This is accepted by the owner until the MCP gateway (F1-T12) routes tool calls through Calango's approval flow. Until then the CLI harness is for trusted workspaces only.
- Streaming and cost are approximate until the `--json` schema is verified against a real run (not done: owner chose no real model calls for this slice).
- Adding Claude Code CLI, OpenCode, Antigravity (`agy`) and the direct API means one provider per CLI, using the same pattern.

## Alternatives rejected
- **Full `Run(ctx, turn, sink)` now.** Duplicates the gateway path and blocks the slice on memory, tools and steps (F1-T07/T11/T12).
- **Temporary `CODEX_HOME` with a copied `auth.json`.** Copies a credential file; the owner chose sharing `~/.codex` instead.
- **Sandbox `read-only`.** Would make Codex unable to edit files; owner chose `workspace-write` after the risk was shown.
- **Parsing `--json` events now.** The schema is not verified; a guessed parser would fail silently.
