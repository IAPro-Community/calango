# Documentation index — Calango

## Control (always read, in this order)

| File | Purpose |
|---|---|
| [HANDOFF.md](HANDOFF.md) | where the work stopped, next step |
| [CONTEXT.md](CONTEXT.md) | current project state |
| [SPECS/ACTIVE.md](SPECS/ACTIVE.md) | contract of the task in progress |
| [SPECS/ROADMAP.md](SPECS/ROADMAP.md) | phases and numbered tasks |
| [TODO_V1.md](TODO_V1.md) | everything still needed for a working V1, by phase |
| [../CONTRIBUTING.md](../CONTRIBUTING.md) | how to contribute (public) |
| [../LICENSE](../LICENSE), [../COMMERCIAL.md](../COMMERCIAL.md), [../CLA.md](../CLA.md), [../TRADEMARKS.md](../TRADEMARKS.md) | license, commercial license, contributor agreement and trademarks |
| [WORKLOG.md](WORKLOG.md) | short history of what was done |
| [VERIFY.md](VERIFY.md) | what was verified |

## Product

| File | Purpose |
|---|---|
| [PLANO.md](PLANO.md) | full overview: vision, modules, roadmap |
| [COMPETITIVE_LANDSCAPE.md](COMPETITIVE_LANDSCAPE.md) | competitors, current sources and implications for V1 |
| [ORQUESTRADOR_MAESTRO.md](ORQUESTRADOR_MAESTRO.md) | embedded Orquestrador Maestro, updated through npm |
| [PRODUCT_HYBRID.md](PRODUCT_HYBRID.md) | PC + VPS + phone model, connections |
| [MONETIZATION.md](MONETIZATION.md) | plans, add-ons, billing |
| [BRAND/BRAND.md](BRAND/BRAND.md) | logo, palette, typography, motion |

## Engineering

| File | Purpose |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | layers and packages |
| [DATA_MODEL.md](DATA_MODEL.md) | SQLite tables and events |
| [SECURITY.md](SECURITY.md) | threat model and defenses |
| [CONVENTIONS.md](CONVENTIONS.md) | code standards for Go, TS, Dart, tests |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Docker, VPS, TLS, Tailscale |
| [TECHNICAL_DELIVERY.md](TECHNICAL_DELIVERY.md) | technical flow, CI, release, migrations and rollback |
| [../api/openapi.yaml](../api/openapi.yaml) | API contract (source of the TS and Dart SDKs) |
| [../packages/tokens/tokens.json](../packages/tokens/tokens.json) | design tokens (source of CSS and ThemeData) |
| [../packages/i18n/messages.json](../packages/i18n/messages.json) | interface catalog in en/pt-BR/es, source of the TypeScript and ARB files |

## Decisions (ADR)

| ADR | Decision |
|---|---|
| [0001](ADR/0001-nucleo-em-go.md) | Core in Go, single `calango` binary |
| [0002](ADR/0002-sqlite-por-no.md) | SQLite per node, no cgo |
| [0003](ADR/0003-desktop-electron.md) | Desktop in Electron with `calango` as a child process |
| [0004](ADR/0004-mobile-flutter.md) | Mobile in Flutter; sharing through the contract |
| [0005](ADR/0005-transporte-tailscale.md) | Tailscale as the initial transport |
| [0006](ADR/0006-codigo-original.md) | Original code; references for reading only |
| [0007](ADR/0007-gateway-de-modelo.md) | Single model gateway with budget |
| [0008](ADR/0008-replicacao-por-eventos.md) | Node-to-node replication by events; code through git |
| [0009](ADR/0009-orquestrador-maestro.md) | Orquestrador Maestro embedded, updated through the official npm channel |
| [0010](ADR/0010-licenca-agpl-dupla.md) | Open source under AGPL-3.0, with commercial license, CLA and trademark policy |
| [0011](ADR/0011-reuso-agenticos.md) | Port Apache-2.0 code from AgenticOS into Go, TypeScript and Dart; amends ADR 0006 for that source |
| [0012](ADR/0012-vault-envelope.md) | Vault stores envelope-encrypted values in SQLite; master key from the OS keyring or a deployment secret |

## Module specs (`SPECS/modules/`)

| Module | Phase |
|---|---|
| [agents](SPECS/modules/agents.md) | 1 |
| [conversations](SPECS/modules/conversations.md) | 1 |
| [config](SPECS/modules/config.md) | 0 |
| [harness](SPECS/modules/harness.md) | 1 |
| [gateway](SPECS/modules/gateway.md) | 1 |
| [maestro](SPECS/modules/maestro.md) | 1 (install), 3 (update) |
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

`site/index.html`: static product presentation page. Interface in English (default), Brazilian Portuguese and Spanish.

## Governance

- [Governance policy](../GOVERNANCE.md)
- [Contribution workflow](../CONTRIBUTING.md)
- [Governance bootstrap and administrative verification](GOVERNANCE_BOOTSTRAP.md)
- [GitHub Issues](https://github.com/IAPro-Community/calango/issues) — operational work; legacy TODO/roadmap pending reconciliation.

- [Auditoria retroativa da main](AUDITORIA_RETROATIVA.md) — mapeamento de entregas históricas e validações pendentes.

- [Planejamento operacional](PLANEJAMENTO_PRODUTO.md) — mapa inicial de épicos, issues e dependências.
