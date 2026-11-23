# Handoff

Atualizado: 2026-11-23.

## Estado

- Go 1.27.4 instalado em `~/.local/opt/go` (link em `~/.local/bin/go`). Para usar em shell novo: `export PATH="$HOME/.local/bin:$PATH"`.
- `go vet ./...`, `go test ./...` (4 testes) e `gofmt` passam. `calango` sobe e `GET /healthz` responde 200.
- Documentação completa para o próximo agente: comece por `AGENTS.md` e `DEV/INDEX.md`.
- O TODO da V1 está em `DEV/TODO_V1.md`.
- Site: `python3 -m http.server 4377 --bind 127.0.0.1 --directory site`.

## Repositório público

- Licença AGPL-3.0 com licença comercial (ADR 0010). Arquivos de comunidade prontos.
- Git local na `main`, sem commit. Publicar em `IAPro-Community/calango` só depois de revisão e pedido explícito.
- Antes de publicar: preencher nome legal do mantenedor no `CLA.md` e os contatos de e-mail.
- Docker: validado. Os dois builds passam e os healthchecks respondem 200 (ver `DEV/VERIFY.md`). A rede bridge do Docker não funciona nesta máquina; os testes de run usaram `--network host`.

## Próximo passo

1. Dono: revisão jurídica de `CLA.md`, `COMMERCIAL.md` e `TRADEMARKS.md`; preencher nome legal do mantenedor e contatos (D1b).
2. Dono: testar `docker run -p 8080:8080 calango-site` num ambiente com rede bridge funcionando, para confirmar o mapeamento de porta.
3. Dono: decidir D3 (primeiro harness). Desbloqueia `F1-T01` (`internal/store` com SQLite).
4. Sem bloqueio: `F0-T04` (CI, já escrito em `.github/workflows/ci.yml`, falta validar no GitHub) e `F0-T05` (config).
5. Publicar em `IAPro-Community/calango` somente com pedido explícito.

## Observações

- `sudo` exige senha nesta máquina; a instalação do Go foi feita sem ele.
- O Maestro ainda não foi testado: falta confirmar os pontos de `DEV/ORQUESTRADOR_MAESTRO.md` (destino de instalação, saída JSON, `update` não interativo).
- A revisão visual completa do site ficou pendente.
