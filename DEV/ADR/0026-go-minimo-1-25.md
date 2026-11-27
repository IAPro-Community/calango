# ADR 0026 — Go mínimo 1.25 para o go-sdk MCP v1.4.1

Status: proposto · issue #193 · substitui a versão mínima de Go registrada no ADR 0018. Precisa de decisão do dono do projeto, como a de 2026-11-23 que fixou o Go 1.23.

## Contexto

O ADR 0018 adotou o `github.com/modelcontextprotocol/go-sdk` v1.3.0, que roda em Go 1.23; o `DEV/HANDOFF.md` registra que versões mais novas só entram com nova decisão. O Dependabot abriu alertas altos no go-sdk que só fecham no v1.4.1 (os intervalos afetados vão até `<= 1.4.0`). O v1.4.1 declara `go 1.25`, então o `go.mod` do Calango precisa subir junto. O #189 tentou o bump e quebrou o CI, que roda em Go 1.23.

## Alternativas avaliadas

- **Ficar no v1.3.0 e aceitar os alertas:** mantém o gateway MCP exposto a falhas conhecidas da biblioteca que fala com o Codex em cada turno. Rejeitado.
- **v1.3.1 ou v1.4.0:** fecham só parte dos alertas; o v1.4.0 já exige Go 1.24. Rejeitado.
- **Fork ou patch local do v1.3.x:** custo de manutenção alto para uma biblioteca oficial. Rejeitado.
- **Subir o Go mínimo para 1.25 e o go-sdk para v1.4.1:** escolhido.

## Decisão

1. `go.mod` declara `go 1.25.0`. O `go mod tidy` sobe junto `golang.org/x/sys` (v0.40.0), `golang.org/x/oauth2` (v0.34.0) e traz `github.com/segmentio/encoding` e `github.com/segmentio/asm` (MIT), linkados no binário.
2. CI (`setup-go`), a imagem de build (`golang:1.25-alpine`) e os guias passam a pedir Go 1.25+. A CI nativa já lê a versão do `go.mod`.
3. `NOTICE` e `third_party/go/` incluem os módulos novos; `npm run licenses:check` confere.

## Consequências

- Quem compila precisa de Go 1.25 ou mais novo. Com `GOTOOLCHAIN=auto` (padrão do Go) o toolchain certo é baixado sozinho; com `GOTOOLCHAIN=local` é preciso instalar.
- O Go 1.25 já saiu da janela de suporte do projeto Go (que cobre as duas versões mais recentes). Subir a CI para uma versão suportada, mantendo o mínimo em 1.25, fica para uma decisão separada.
- O staticcheck segue fixado em `2025.1.1`; se ele não entender a linguagem do 1.25, a subida dele entra neste mesmo PR.
