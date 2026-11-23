# Módulo: cloud (Calango Cloud)

Fases 2–3 · `cmd/calango-cloud` (Go) · Postgres

## Objetivo
Serviços nossos, mínimos: conta, dispositivos, licença, billing, push, updates. **Nunca** conversas, arquivos ou chaves de modelo.

## Serviços
| Serviço | Rotas | Fase |
|---|---|---|
| account | `/v1/auth/*`, `/v1/me`, `/v1/nodes` | 2 |
| push | `/v1/push/register`, `/v1/push/send` | 2 |
| license | `/v1/license/refresh` | 3 |
| billing | `/v1/billing/checkout`, webhooks Stripe/Apple/Google | 3 |
| updates | `/v1/updates/{channel}/{platform}` | 1 |

## Regras
- Login por e-mail com link mágico e OAuth (Google, Apple, GitHub).
- Telemetria só com opt-in, sem conteúdo.
- Um binário, serviços como módulos; deploy em container.
- Banco Postgres; migrações imutáveis.

## Aceite
- Auditoria de dados: nenhuma tabela da Cloud tem coluna com texto de conversa.
