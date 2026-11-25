# Module: harness

Phase 1 · Package `internal/harness`

## V1 status (2026-11-23)
- **Done (F1-T05 partial):** `internal/harness.Codex` is a `gateway.Provider` named `codex` ([ADR 0013](../../ADR/0013-harness-cli-como-provedor.md)). It runs `codex exec` isolated (`--ephemeral --ignore-user-config --ignore-rules`), prompt on stdin, final message via `-o`. `Available` checks the binary (`exec.LookPath`) and `codex login status`. Signed out → `ErrSignedOut` → `failed/harness_signed_out` (`core.ErrCodeHarnessSignedOut`, mapped in `conversations`).
- **Deviation from the interface above:** V1 uses the gateway `Provider` contract (`Name`, `Complete`), not `Run(ctx, turn, sink)`. `Turn`, steps and the MCP tool list come with F1-T07/T11/T12.
- **Not verified:** the `--json` event schema (usage is estimated, one chunk per reply); whether `~/.codex/AGENTS.md` loads under `--ignore-user-config`; a real Codex run (only a fake `codex` script was used in tests).
- **Known limits:** sandbox `workspace-write` (commands and writes run without Calango approval until F1-T12); cancellation kills the CLI but not its child processes. Isolation: a non-default account runs with its own `CODEX_HOME` (ADR 0016); `default` shares `~/.codex` (ADR 0013).
- **Wired:** the daemon resolves `codex/<label>` accounts through `gateway.Config.Resolve` (binary `codex` on PATH, workdir `<data-dir>/workspace`); agents select them in their provider chain (ADR 0016).
- **Next V1 harnesses:** Claude Code CLI, OpenCode, Antigravity (Google, CLI `agy`, auth through OS keyring/OAuth, `GEMINI_API_KEY`, or ADC with `AGY_ADC_AUTH=true`), and the OpenAI API. The direct Anthropic API is implemented in `internal/anthropic` (F1-T04). Each is one `gateway.Provider` per ADR 0013.


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

## Integração SDK (issue #128)

O primeiro recorte usa tipos canônicos de provider e classificação HTTP do IAPro-SDK, além do gate compartilhado de fallback da cadeia. Consulte [ADR 0020](../../ADR/0020-iapro-sdk.md). Autenticação, execução, quota real, continuidade, coordenação e HUD permanecem no roadmap de integração; não há execução automática fora da cadeia ou mudança de permissões.

### Runtime Codex compartilhado (SDK, segundo recorte)

Execução JSONL via IAPro-SDK e launch controlado pelo Calango (ADR 0020). Mensagens completas são entregues progressivamente; contadores terminais medidos prevalecem, estimativas são explícitas e falhas sem contagem usam a reserva do gateway. CLI real e versões compatíveis exigem homologação autorizada. Sem mudança na política MCP, sem persistência de sessão ou seleção automática por quotas.
