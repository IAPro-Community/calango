# Módulo: approvals

Fase 1 · Pacote `internal/approvals` · Tabelas `approvals`, `approval_rules`, `audit_log`

## Objetivo
Nenhuma ação sensível acontece sem uma pessoa aprovar.

## Regras
- Ações sensíveis (lista fixa no código): enviar mensagem/e-mail, apagar, publicar, pagar, deploy, merge, comando de shell fora da lista segura, escrita via conector com acesso `write`.
- O run pausa em `waiting_approval` até a decisão. Expira em 24 h (configurável) → `expired`, o run segue sabendo que não foi aprovado.
- Cartão mostra **ação, alvo e argumentos exatos**.
- Recibo depois: `completed`, `failed` (com motivo) ou `not_run`.
- "Aprovar sempre" grava regra por agente e tipo de ação.
- Aprovações de um mesmo run são apresentadas na ordem em que foram pedidas (`seq`).
- O agente nunca decide aprovação. Decisão exige dispositivo pareado de uma pessoa.
- Toda decisão vai para `audit_log`.

## API
`GET /v1/approvals?status=pending`, `POST /v1/approvals/{id}/decide` `{ decision: "approve"|"deny", remember: bool }`.

## Eventos
`approval.requested`, `approval.decided`.

## Aceite
- Ação aprovada que falha mostra `failed` com motivo, não "aprovado".
- Com regra "aprovar sempre", a mesma ação não gera cartão, mas gera entrada de auditoria.
