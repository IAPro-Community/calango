# ADR 0004 — Mobile em Flutter; compartilhamento por contrato

Status: aceita · 2026-11-23

## Contexto
Apps iOS e Android para controle remoto, aprovações, conversas e notificações.

## Decisão
Flutter (Dart). O compartilhamento com o desktop (React/TS) é por contrato, não por código:
- `api/openapi.yaml` gera `packages/sdk-ts` e `packages/sdk-dart`.
- `packages/tokens/tokens.json` gera CSS e `ThemeData`.
- `packages/i18n` gera JSON (desktop) e ARB (Flutter).

## Consequências
- Telas duplicadas. Mitigação: o mobile tem escopo menor e não replica a ADE.
- Qualquer mudança de API começa no OpenAPI; SDK escrito à mão é proibido.
