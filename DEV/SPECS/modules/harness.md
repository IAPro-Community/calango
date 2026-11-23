# Módulo: harness

Fase 1 · Pacote `internal/harness`

## Objetivo
Executar um turno de agente usando um "cérebro": API direta de modelo ou CLI de agente (Claude Code, Codex, Gemini CLI, Grok CLI, OpenCode).

## Interface (Go)
```go
type Harness interface {
    Name() string
    Available(ctx context.Context) (Status, error)   // instalado? logado?
    Run(ctx context.Context, turn Turn, sink StepSink) (Result, error)
}
```
`Turn` leva instruções, memória, histórico resumido, lista de ferramentas (URL do gateway MCP local com token do turno) e esforço. `StepSink` recebe os passos em streaming.

## Regras
- O CLI roda **isolado da configuração pessoal** da pessoa: diretório de config temporário, sem ferramentas próprias, sem sessão salva. O contexto é montado pelo Calango a cada turno.
- A única fonte de ferramentas do CLI é o gateway MCP local do Calango, com uma URL secreta por turno listando só as ferramentas daquele agente.
- Conteúdo externo (página, e-mail, arquivo) entra no prompt marcado como dado.
- Tokens: usar o que o CLI informa; se não informar, estimar e marcar `estimated`.
- **Login que expira** é o problema mais comum nas referências: `Available` detecta, e o run falha com `harness_signed_out` e uma mensagem clara de como relogar. Nunca falhar em silêncio nem com resposta vazia.

## Aceite
- Harness fake nos testes cobre streaming, erro, cancelamento.
- Com o CLI deslogado, o run termina em `failed/harness_signed_out` e a interface mostra o botão de relogar.
