# Worklog

## 2026-11-22 — fundação (fase 0)

- Varredura de referências: Dot (encerrado), Grok Bot, Tucano, agentes.chat. Leitura de README, CHANGELOG, ADRs, issues e PRs abertos. Clones ficam fora do projeto, só como referência.
- Decidido: núcleo em Go, um binário `calangod`; SQLite puro Go; Wails para desktop; mobile nativo consumindo a API (escolha Flutter vs React Native em aberto).
- Criado esqueleto: `go.mod`, `internal/core` (agente com orçamento, agendador de rotinas), `internal/api` (`/healthz`), `cmd/calangod`.
- Testes escritos, não executados: Go não está instalado nesta máquina.
- Decidido: transporte entre nós por Tailscale na fase inicial (avaliar tsnet para embutir). Relay próprio depois. Registrado em `DEV/PRODUCT_HYBRID.md`.
- 2026-11-23: plano completo escrito em `DEV/PLANO.md` (visão, topologia, stack, 17 módulos, segurança, dados, monetização, distribuição, monorepo, roadmap em 8 fases, lições das referências).
- 2026-11-23: desktop passa a ser Electron (antes Wails). Binário renomeado de `calangod` para `calango` (`cmd/calango`), em código, Docker e docs.
- 2026-11-23: mobile decidido como Flutter. Plano ajustado: OpenAPI gera SDK TypeScript e Dart; tokens e traduções com fonte única; monorepo ganha `api/`, `sdk-dart/` e `tokens/`.
- 2026-11-23: documentação técnica completa para outro agente construir: `AGENTS.md` na raiz, `DEV/INDEX.md`, `CONTEXT.md`, `CONVENTIONS.md`, `SECURITY.md`, `DATA_MODEL.md`, 8 ADRs, `SPECS/ROADMAP.md` com tarefas numeradas (F0–F7), 20 specs de módulo em `SPECS/modules/`, `api/openapi.yaml` (rascunho v0.1), `packages/tokens/tokens.json` (paleta amostrada da logo), `DEV/BRAND/BRAND.md`.
- 2026-11-23: site de apresentação em `site/index.html` (estático): hero com parede de post-its e tile de vidro, demo do produto movida por relógio único (5 cenas, pausar/arrastar/pular), diagrama PC↔VPS↔celular animado, recursos, ADE com código e terminal digitando, segurança, planos, plataformas, roadmap, FAQ. Claro/escuro e `prefers-reduced-motion`. Logo copiada para `DEV/BRAND/logo-original.png` e redimensionada em `site/assets/`.
- 2026-11-23: Go 1.27.4 instalado sem sudo em `~/.local/opt/go` (`~/.local/bin/go`), SHA-256 conferido contra go.dev. `go vet`, `go test` (4 testes) e `gofmt` passam. `calango` sobe e `/healthz` responde 200.
- 2026-11-23: Orquestrador Maestro (Apache-2.0) incorporado por ADR 0009: consumido pelo npm `@iapro/orquestrador-maestro-cli` na linha V1 (1.1.1), atualização com cartão ou automática (padrão desligada), `verify` e rollback. Documentado em `DEV/ORQUESTRADOR_MAESTRO.md`, `SPECS/modules/maestro.md`, `NOTICE`. ADR 0006 ganhou exceção só para o Maestro.
- 2026-11-23: `DEV/TODO_V1.md` criado com todas as tarefas da V1 (fases 0–6 do roadmap + Maestro + transversais) e decisões D1–D6 que bloqueiam partes da entrega.
- 2026-11-23: dono decidiu que o código do Calango será aberto (D1). Falta escolher a licença específica (D1a); recomendação: Apache-2.0. Nenhum `LICENSE` foi criado ainda.
- 2026-11-23: licença decidida: AGPL-3.0 + licença comercial + CLA + política de marca (ADR 0010). Criados `LICENSE` (texto oficial, gnu.org), `COMMERCIAL.md`, `CLA.md` (rascunho para revisão), `TRADEMARKS.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `.gitignore`, `.dockerignore`, `.github/workflows/ci.yml`.
- 2026-11-22: revisão do site (`site/index.html`). Contraste medido no navegador: títulos e eyebrows verdes, botão principal, números do roadmap, cards de ADE e tags no escuro estavam abaixo de AA. Corrigido com tokens de texto (`--moss-text`, `--btn`, `--num`, `--warn-text`, `--on-tone`). Bug grave corrigido: título da seção ADE ficava invisível no tema escuro. Copy enxugada (leads, features de 12 para 8, segurança de 6 para 4 cards). Efeitos novos: fita com capacidades, barra de progresso, marcador que se desenha nos títulos, inclinação do palco com o mouse, tilt e hover nos cards, tudo sob `prefers-reduced-motion`.
- 2026-11-22: tentativa de validar Docker bloqueada pelo grupo `docker`. Depois que o dono adicionou o usuário ao grupo, os dois builds passaram (site e nó). Healthchecks do site e do nó responderam 200 com `--network host`. A rede bridge não funciona nesta máquina (`operation not supported`).
- 2026-11-23: Dockerfile da raiz virou o **site** (nginx não-root, porta 8080, healthcheck, CSP). Dockerfile do nó movido para `deploy/Dockerfile.node`. Seção "Contribua com a V1" adicionada ao site.
- 2026-11-23: git local iniciado na `main`, **sem commit**. Nada publicado.
- Modelo de implantação: servidor na VPS da pessoa via Docker, clientes conectam por TLS. Documentado em `DEV/DEPLOYMENT.md`. Dockerfile, compose e Caddyfile criados, não testados.
