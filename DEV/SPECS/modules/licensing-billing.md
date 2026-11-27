# Module: licensing-billing

Phase 3 · `cmd/calango-cloud` (services `license`, `billing`) + `internal/entitlements` on the node

## Goal
Charge for the plans and unlock paid features without making the node depend on the internet.

## License
- The Cloud issues an Ed25519-signed token: `{ account_id, plan, features[], nodes_max, exp }`, valid for 7 days, renewed automatically.
- The node validates the signature with the public key embedded in the binary. Without network, it is valid until `exp` + a 7-day tolerance.
- When it expires: the node **downgrades to Free** (paid features turn off, warnings appear). It never stops starting and never deletes data.

## Entitlements (in the core)
```go
type Entitlements struct {
    Plan        string
    VPSNodes    int
    Routines    int  // -1 = unlimited
    Projects    int
    RemoteScreen bool
    Push        bool
}
```
Checked in the service, not in the interface.

## Billing
- Stripe for web and desktop (subscription + credits).
- App Store and Google Play for purchases on the phone (via RevenueCat or direct integration).
- One source of truth for the plan in the Cloud, fed by webhooks from all three sources.

## Acceptance
- A VPS node offline for 10 days with a 7-day license keeps starting in Free mode and warns.
- A tampered token is refused.

## Implementação no node (issue #75)

Pacote `internal/license`; billing (#76) e `Entitlements` (#77) ficam fora deste recorte.

- **Formato:** `calango-license-v1.<payload>.<assinatura>`, ambos em base64url sem padding. O payload é JSON `{ v, account_id, plan, features[], nodes_max, iat, exp }` (tempos em segundos Unix; `nodes_max` = -1 para ilimitado; `plan` ∈ `pro`, `team`, `business`). A assinatura Ed25519 cobre `calango-license-v1.<payload>`, então não pode ser reaproveitada em outro formato. Token limitado a 8 KiB; `v` diferente de 1, plano desconhecido ou campos inválidos recusam a licença mesmo com assinatura válida.
- **Chave pública:** embutida no binário de release via `-ldflags "-X calango/internal/license.embeddedPublicKey=<base64>"`. Quando há chave embutida, `CALANGO_LICENSE_PUBLIC_KEY` é ignorada (com aviso no log), para que configuração não troque o emissor. Builds de desenvolvimento, sem chave embutida, usam `CALANGO_LICENSE_PUBLIC_KEY`; sem nenhuma chave, toda licença é `invalid/no_public_key` e o node roda Free. A chave privada fica só no serviço de licenças, nunca neste repositório.
- **Arquivo:** `CALANGO_LICENSE_FILE` (padrão `<CALANGO_DATA_DIR>/calango.license`). É relido a cada consulta, então uma licença renovada vale sem reiniciar. O node só lê o arquivo: nunca escreve, renomeia ou apaga nada.
- **Estados** (relógio injetável): `none` (sem arquivo), `active` (`agora < exp`), `grace` (`exp ≤ agora < exp + 7 dias`; o plano continua), `expired` (tolerância encerrada) e `invalid` (`malformed`, `signature`, `claims`, `no_public_key`, `unreadable`). Em `expired` e `invalid` o plano efetivo é `free`. Nenhum estado impede o node de iniciar ou apaga dados; os claims verificados continuam visíveis após a expiração para o aviso.
- **Leitura:** `GET /v1/license` (permissão `read`) devolve estado, plano efetivo, motivo e os campos da licença. O daemon registra no log o estado ao iniciar e a cada mudança (checagem horária).
- **Ferramenta de dev:** `go run ./cmd/calango-devlicense keygen -out dev.key` imprime a chave pública e grava a semente privada (0600, recusa sobrescrever); `issue -key dev.key -plan pro -days 7` emite uma licença de teste (`-days` negativo gera licença já vencida). Não entra na imagem do node.
- **Pendências:** renovação automática junto ao serviço de licenças (Cloud), aviso na interface, aplicação do plano pelos `Entitlements` (#77) e limite de `nodes_max` (não verificável offline).
