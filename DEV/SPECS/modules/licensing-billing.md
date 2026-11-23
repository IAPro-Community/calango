# Módulo: licensing-billing

Fase 3 · `cmd/calango-cloud` (serviços `license`, `billing`) + `internal/entitlements` no nó

## Objetivo
Cobrar os planos e liberar recursos pagos sem tornar o nó dependente de internet.

## Licença
- A Cloud emite um token assinado Ed25519: `{ account_id, plan, features[], nodes_max, exp }`, validade 7 dias, renovado automaticamente.
- O nó valida a assinatura com a chave pública embutida no binário. Sem rede, vale até `exp` + 7 dias de tolerância.
- Expirou: o nó **rebaixa para Free** (recursos pagos desligam, avisos aparecem). Nunca para de subir, nunca apaga dados.

## Entitlements (no núcleo)
```go
type Entitlements struct {
    Plan        string
    VPSNodes    int
    Routines    int  // -1 = ilimitado
    Projects    int
    RemoteScreen bool
    Push        bool
}
```
Checados no serviço, não na interface.

## Billing
- Stripe para web e desktop (assinatura + créditos).
- App Store e Google Play para compra no celular (via RevenueCat ou integração direta).
- Uma fonte de verdade do plano na Cloud, alimentada por webhooks das três fontes.

## Aceite
- Nó VPS offline por 10 dias com licença de 7: continua subindo em modo Free e avisa.
- Token adulterado é recusado.
