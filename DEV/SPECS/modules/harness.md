# Module: harness

Phase 1 · Package `internal/harness`

## Goal
Run one agent turn using a "brain": a direct model API or an agent CLI (Claude Code, Codex, Gemini CLI, Grok CLI, OpenCode).

## Interface (Go)
```go
type Harness interface {
    Name() string
    Available(ctx context.Context) (Status, error)   // installed? signed in?
    Run(ctx context.Context, turn Turn, sink StepSink) (Result, error)
}
```
`Turn` carries instructions, memory, a summarized history, the tool list (local MCP gateway URL with the turn's token) and effort. `StepSink` receives the steps as a stream.

## Rules
- The CLI runs **isolated from the person's personal configuration**: temporary config directory, no own tools, no saved session. Calango builds the context at every turn.
- The only tool source for the CLI is Calango's local MCP gateway, with a secret URL per turn listing only that agent's tools.
- External content (page, email, file) enters the prompt marked as data.
- Tokens: use what the CLI reports; if it does not report, estimate and mark `estimated`.
- **Expired sign-in** is the most common problem in the references: `Available` detects it, and the run fails with `harness_signed_out` and a clear message on how to sign in again. Never fail silently or with an empty reply.

## Acceptance
- A fake harness in tests covers streaming, errors and cancellation.
- With the CLI signed out, the run ends in `failed/harness_signed_out` and the interface shows a sign-in button.
