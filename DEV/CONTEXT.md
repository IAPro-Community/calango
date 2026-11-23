# Contexto atual

Atualizado: 2026-11-23.

- **Fase:** 0 (fundação). Esqueleto Go escrito, **não compilado** (Go não instalado na máquina do dono).
- **Existe:** `cmd/calango` (servidor HTTP + agendador), `internal/core` (agente com orçamento, agendador), `internal/api` (`/healthz`), testes de core, Dockerfile, compose, Caddyfile, docs completos, rascunho do OpenAPI, tokens de design, site de apresentação.
- **Não existe:** banco, gateway de modelo, harness, desktop, mobile, cloud.
- **Decidido:** Go, SQLite, Electron, Flutter, Tailscale, código original, nome `calango`.
- **Em aberto:** licença do código do Calango, controle de tela no Pro ou Team, primeiro harness do MVP, domínio, nome do módulo Go.
- **Referências lidas (só leitura, fora do repo):** Tucano v0.2.45 (Apache-2.0, Electron, SQLite) e agentes.chat (AGPL-3.0, Bun/Postgres), incluindo changelog, ADRs, issues e PRs abertos em 2026-11-22.
