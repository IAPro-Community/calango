# Índice da documentação — Calango

## Controle (ler sempre, nesta ordem)

| Arquivo | Para quê |
|---|---|
| [HANDOFF.md](HANDOFF.md) | onde parou, próximo passo |
| [CONTEXT.md](CONTEXT.md) | estado atual do projeto |
| [SPECS/ACTIVE.md](SPECS/ACTIVE.md) | contrato da tarefa em andamento |
| [SPECS/ROADMAP.md](SPECS/ROADMAP.md) | fases e tarefas numeradas |
| [TODO_V1.md](TODO_V1.md) | tudo que falta para a V1 funcional, por fase |
| [../CONTRIBUTING.md](../CONTRIBUTING.md) | como contribuir (público) |
| [../LICENSE](../LICENSE), [../COMMERCIAL.md](../COMMERCIAL.md), [../CLA.md](../CLA.md), [../TRADEMARKS.md](../TRADEMARKS.md) | licença, licença comercial, acordo de contribuidor e marca |
| [WORKLOG.md](WORKLOG.md) | histórico curto do que foi feito |
| [VERIFY.md](VERIFY.md) | o que foi verificado |

## Produto

| Arquivo | Para quê |
|---|---|
| [PLANO.md](PLANO.md) | panorama completo: visão, módulos, roadmap |
| [ORQUESTRADOR_MAESTRO.md](ORQUESTRADOR_MAESTRO.md) | Orquestrador Maestro embutido e atualizado pelo npm |
| [PRODUCT_HYBRID.md](PRODUCT_HYBRID.md) | modelo PC + VPS + celular, conexões |
| [MONETIZATION.md](MONETIZATION.md) | planos, add-ons, cobrança |
| [BRAND/BRAND.md](BRAND/BRAND.md) | logo, paleta, tipografia, movimento |

## Engenharia

| Arquivo | Para quê |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | camadas e pacotes |
| [DATA_MODEL.md](DATA_MODEL.md) | tabelas SQLite e eventos |
| [SECURITY.md](SECURITY.md) | modelo de ameaças e defesas |
| [CONVENTIONS.md](CONVENTIONS.md) | padrões de código Go, TS, Dart, testes |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Docker, VPS, TLS, Tailscale |
| [../api/openapi.yaml](../api/openapi.yaml) | contrato da API (fonte dos SDKs TS e Dart) |
| [../packages/tokens/tokens.json](../packages/tokens/tokens.json) | design tokens (fonte de CSS e ThemeData) |

## Decisões (ADR)

| ADR | Decisão |
|---|---|
| [0001](ADR/0001-nucleo-em-go.md) | Núcleo em Go, binário único `calango` |
| [0002](ADR/0002-sqlite-por-no.md) | SQLite por nó, sem cgo |
| [0003](ADR/0003-desktop-electron.md) | Desktop em Electron com `calango` como processo filho |
| [0004](ADR/0004-mobile-flutter.md) | Mobile em Flutter; compartilhamento por contrato |
| [0005](ADR/0005-transporte-tailscale.md) | Tailscale como transporte inicial |
| [0006](ADR/0006-codigo-original.md) | Código original; referências só para leitura |
| [0007](ADR/0007-gateway-de-modelo.md) | Gateway único de modelo com orçamento |
| [0008](ADR/0008-replicacao-por-eventos.md) | Replicação entre nós por eventos; código por git |
| [0009](ADR/0009-orquestrador-maestro.md) | Orquestrador Maestro embutido, atualizado pelo canal npm oficial |
| [0010](ADR/0010-licenca-agpl-dupla.md) | Código aberto sob AGPL-3.0, com licença comercial, CLA e marca |

## Specs de módulo (`SPECS/modules/`)

| Módulo | Fase |
|---|---|
| [agents](SPECS/modules/agents.md) | 1 |
| [conversations](SPECS/modules/conversations.md) | 1 |
| [harness](SPECS/modules/harness.md) | 1 |
| [gateway](SPECS/modules/gateway.md) | 1 |
| [maestro](SPECS/modules/maestro.md) | 1 (instalação), 3 (atualização) |
| [routines](SPECS/modules/routines.md) | 1 |
| [approvals](SPECS/modules/approvals.md) | 1 |
| [memory](SPECS/modules/memory.md) | 1 |
| [tools-mcp](SPECS/modules/tools-mcp.md) | 1 |
| [vault](SPECS/modules/vault.md) | 1 |
| [desktop-electron](SPECS/modules/desktop-electron.md) | 1 |
| [pairing-auth](SPECS/modules/pairing-auth.md) | 2 |
| [mobile-flutter](SPECS/modules/mobile-flutter.md) | 2 |
| [push](SPECS/modules/push.md) | 2 |
| [nodes-sync](SPECS/modules/nodes-sync.md) | 3 |
| [licensing-billing](SPECS/modules/licensing-billing.md) | 3 |
| [cloud](SPECS/modules/cloud.md) | 2–3 |
| [ade](SPECS/modules/ade.md) | 4 |
| [browser](SPECS/modules/browser.md) | 5 |
| [computer-use](SPECS/modules/computer-use.md) | 5 |
| [channels-teams](SPECS/modules/channels-teams.md) | 6 |

## Site

`site/index.html`: página de apresentação do produto (estática).
