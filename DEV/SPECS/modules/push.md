# Módulo: push

Fase 2 · `cmd/calango-cloud` (serviço `push`) + `internal/push` no nó

## Objetivo
Notificar o celular mesmo com o app fechado, sem que a Calango veja conteúdo.

## Fluxo
1. O app registra o token APNs/FCM na Calango Cloud, associado ao dispositivo.
2. O nó envia ao relay: `{ device_id, kind, opaque_id }`. Nada de texto.
3. O relay envia ao APNs/FCM uma notificação genérica ("Calango precisa de você") com `opaque_id`.
4. O app abre e busca o conteúdo no nó pelo transporte.

## Regras
- Autenticação do nó no relay por chave da conta.
- Limite por plano (Free limitado, Pro ilimitado razoável).
- Agrupamento: várias notificações em 30 s viram uma.

## Aceite
- Inspeção do payload no relay não contém texto de conversa (teste).
