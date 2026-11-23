# ADR 0007 — Single model gateway with budget

Status: accepted · 2026-11-23

## Context
Freemium and spending limits depend on measuring every model call, whether it comes from a direct API or from a harness CLI.

## Decision
Every model call goes through `internal/gateway`:
1. Checks the budget of the agent (and of the routine, if any) before the call. Exceeded → `budget_exceeded` error; no call is made.
2. Executes the call through the provider or harness.
3. Records tokens and cost in `usage` with agent, routine, project and node.
4. Emits `usage.threshold` when passing 80% and 100%.

Harness CLI that does not report tokens: the gateway estimates them and marks `estimated=true`.

## Consequences
- No other package imports a provider SDK directly.
- The same record feeds the usage panel, plan limits and credit billing.
