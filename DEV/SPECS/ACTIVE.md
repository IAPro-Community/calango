# Calango — spec ativa

Status: fase 0 (fundação). Código escrito do zero. Repos Tucano e agentes.chat são referência de leitura apenas: nenhum código, texto ou asset copiado.

## Objetivo

Assistente pessoal que roda no computador e continua na nuvem: executa tarefas, rotinas (crons), canais com pessoas e agentes, navegador próprio, e uma ADE para programar projetos separados. Clientes: desktop, iOS, Android e web.

## Decisões de stack (proposta, a validar)

- Núcleo e servidor: **Go**. Um único binário, `calango`, roda local e na nuvem.
- Banco: **SQLite** via `modernc.org/sqlite` (puro Go, sem cgo, cross-compile fácil). Postgres na nuvem fica para depois, atrás de interface.
- API: HTTP + WebSocket (tempo real).
- Navegador do agente: **chromedp** (Chrome/Chromium via DevTools Protocol).
- Desktop: **Electron** (Windows, macOS, Linux), que inicia o `calango` como processo filho. Decidido em 2026-11-23.
- Mobile: **Flutter** (iOS e Android). Decidido em 2026-11-23. Compartilha com o desktop por contrato: OpenAPI gera SDK TypeScript e Dart; tokens de design e traduções têm fonte única.
- Plano completo: [../PLANO.md](../PLANO.md).

## Fase 0 — escopo desta entrega

- [x] Estrutura do repo, docs DEV/.
- [x] Modelo de agente com orçamento de tokens.
- [x] Agendador de rotinas com relógio injetável (testável).
- [x] Servidor HTTP com `/healthz`.
- [ ] Build e testes verificados: **pendente**, Go não instalado nesta máquina.

## Transporte entre nós (decidido)

Tailscale na fase inicial, depois relay próprio. Detalhes em [PRODUCT_HYBRID.md](PRODUCT_HYBRID.md).

## Fora do escopo agora

Canais, integração com modelos, navegador, ADE, clientes mobile e desktop, sincronização nuvem↔local.

## Critério de aceite da fase 0

`go test ./...` e `go vet ./...` passam; `calango` sobe e responde `GET /healthz` com 200.
