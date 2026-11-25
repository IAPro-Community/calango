# Process configuration

## Goal

Load environment configuration in a single package and reject invalid values before starting the server or background tasks.

## Current contract

| Variable | Default | Validation |
|---|---|---|
| `CALANGO_ADDR` | `127.0.0.1:7317` | `host:port` address; host required; numeric port from 1 to 65535. Use `0.0.0.0` only when the node must accept network connections, such as in the container behind a TLS proxy. |
| `CALANGO_VAULT_KEY` | unset | Vault master key: base64 of 32 bytes, for hosts without an OS keyring. Mutually exclusive with `CALANGO_VAULT_KEY_FILE`. Secret: never logged; errors do not echo it. |
| `CALANGO_VAULT_KEY_FILE` | unset | Path to a file with the same base64 key (for example a Docker secret kept outside the data volume). Mutually exclusive with `CALANGO_VAULT_KEY`. |
| `CALANGO_VAULT_KEY_VERSION` | `1` | Version of the current explicit key (`CALANGO_VAULT_KEY` or `_FILE`). Integer ≥ 1. Above 1 needs an explicit current key; the OS keyring stays at version 1. See the ADR 0012 amendment. |
| `CALANGO_VAULT_KEY_PREVIOUS` | unset | Old master key during rotation: base64 of 32 bytes. Needs an explicit current key and version ≥ 2. Mutually exclusive with `CALANGO_VAULT_KEY_PREVIOUS_FILE`. Remove after the rewrap is logged. Secret: never logged; errors do not echo it. |
| `CALANGO_VAULT_KEY_PREVIOUS_FILE` | unset | Path to a file with the same old key. Mutually exclusive with `CALANGO_VAULT_KEY_PREVIOUS`. |
| `CALANGO_TRUSTED_PROXIES` | unset | Comma-separated IP addresses or CIDR ranges of reverse proxies. When the TCP peer is listed, the `/v1/pair` limit uses the right-most `X-Forwarded-For` entry that is not a listed proxy. Empty trusts none. Malformed entries fail startup; errors name the variable only. |

| `CALANGO_CAPACITY_INTERVAL` | unset (disabled) | Opt-in quota monitor for dedicated Codex profiles only; duration 30s–1h, also defines the maximum snapshot TTL. Requires exclusive profile use. |
| `CALANGO_CAPACITY_ROUTING_ALLOWED` | unset (false) | Explicit operator opt-in for dedicated profiles, provider terms and verified zero-marginal-cost tariffs; requires capacity interval and tariff file. |
| `CALANGO_PRICE_TABLE_FILE` | unset | Absolute path to a versioned JSON tariff file in USD micros per million tokens. Invalid configured files stop startup. Unlisted account/model combinations remain unknown. See below. |

`internal/config.Load` is the only point that reads `CALANGO_*` variables. New variables must come here with a default, validation, usage and documentation. Secrets must not be placed in environment variables when the system vault is supported. The one exception is the VPS vault master key (`CALANGO_VAULT_KEY`, `CALANGO_VAULT_KEY_FILE`, and the `_VERSION`, `_PREVIOUS` and `_PREVIOUS_FILE` variables during rotation), because a VPS has no OS vault (ADR 0012).

## Invariants

- The default listens only on loopback.
- Invalid configuration ends the process with an error before the scheduler starts.
- A public address is opt-in and must be protected by the topology documented in `DEV/DEPLOYMENT.md`.
- Errors do not print credential values. `Config` can hold the vault master key; never print or log a `Config`.

## Verification

- Unit tests cover the default, an explicit address, invalid formats/ports, and the vault key settings (mutually exclusive sources).
- `go test ./...` and `go vet ./...` run in CI.

## Tabela de preços (#60)

O arquivo é opcional; não contém credenciais. Exemplo sintético, **não usar como preço real**:

```json
{"version":"example-not-a-real-price","currency":"USD","models":{"anthropic/default/example-model":{"input_per_million":1000000,"output_per_million":2000000}}}
```

A chave usa exatamente o identificador do provider/conta seguido do modelo enviado ao gateway. Não há fallback de tarifa entre contas ou providers. Os valores são inteiros em milionésimos de USD por milhão de tokens. Ambos são obrigatórios; zero é válido somente se configurado explicitamente. Sem arquivo ou entrada correspondente, custo permanece desconhecido. Alterações exigem reinício e nova versão de tabela; não reprecificam o histórico. Preço configurado multiplicado por tokens medidos não comprova cobrança do fornecedor, especialmente em planos de assinatura, cache e descontos; não cadastrar tarifa sem validar a modalidade da conta. Homologação de custo real continua em #60.

## Monitor de capacidade (#130)

Ative em ambiente de homologação com `CALANGO_CAPACITY_INTERVAL=5m`. Não habilite para homes usados por outros processos, nem compartilhe credenciais. O daemon consulta somente perfis Codex dedicados, com identidade local e atribuição confirmada pelo protocolo do SDK. A consulta usa o app-server e não inicia uma conversa. Auth sem metadados verificáveis permanece UNKNOWN; Anthropic API e o Codex default são UNSUPPORTED. A projeção é read-only, não concede autorização financeira nem habilita troca automática. O botão de atualização respeita TTL e backoff; a UI lê o snapshot do daemon. Reinício descarta evidências de quota. Homologação de identidade, versão do app-server e lock fora do processo continua pendente.

## Roteamento por capacidade (#131)

O modo experimental exige autorização operacional explícita (`CALANGO_CAPACITY_ROUTING_ALLOWED=true`), confirmação dos termos e modalidade de conta, tabela de tarifas versionada e opt-in por agente (`capacity_routing`). A configuração padrão mantém o fallback explícito existente. Para troca automática, apenas Codex dedicado, mesmo modelo confirmado pelo snapshot, quota vigente, identidade verificada e tarifa marginal explicitamente zero. Não inferir tarifa zero de assinatura ou quota; não usar para contornar restrições de provider. Sem esses dados, interrompe antes da chamada. Não permite continuidade entre contas, nem retry depois de uma chamada. A reserva SQLite persiste até o término e registro do ledger; falhas ambíguas mantêm a reserva e exigem reconciliação do fence, sem takeover por TTL. A prova de quiescência do processo e seus descendentes, recuperação operacional após crash, coordenação entre produtos ou outros processos e providers reais permanecem gates de homologação. Não habilitar em produção antes dos gates #113/#130/#131.
