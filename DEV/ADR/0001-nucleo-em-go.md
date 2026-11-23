# ADR 0001 — Núcleo em Go, binário único `calango`

Status: aceita · 2026-11-23

## Contexto
O mesmo agente precisa rodar no PC (Windows, macOS, Linux) e na VPS (Linux, Docker), com baixo consumo e distribuição simples.

## Decisão
Todo o núcleo é Go, compilado num binário único chamado `calango` (entrada em `cmd/calango`). PC e VPS rodam o mesmo binário; muda só a configuração.

## Alternativas
- Node/TypeScript (como as referências): mais próximo do desktop, mas pior em distribuição e consumo na VPS.
- Rust: ótimo desempenho, curva maior e iteração mais lenta.

## Consequências
- Cross-compile trivial (`GOOS`/`GOARCH`), imagem Docker mínima.
- Desktop (TS) e mobile (Dart) falam com o núcleo só pela API; nada de lógica duplicada.
- Bibliotecas de IA em Go são menos maduras: o núcleo fala com modelos via HTTP direto e via CLIs de harness.
