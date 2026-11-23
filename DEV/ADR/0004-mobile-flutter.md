# ADR 0004 — Flutter mobile; sharing by contract

Status: accepted · 2026-11-23

## Context
iOS and Android apps for remote control, approvals, conversations and notifications.

## Decision
Flutter (Dart). Sharing with the desktop (React/TS) happens by contract, not by code:
- `api/openapi.yaml` generates `packages/sdk-ts` and `packages/sdk-dart`.
- `packages/tokens/tokens.json` generates CSS and `ThemeData`.
- `packages/i18n` generates JSON (desktop) and ARB (Flutter).

## Consequences
- Duplicated screens. Mitigation: the mobile app has a smaller scope and does not replicate the ADE.
- Any API change starts in the OpenAPI; a hand-written SDK is forbidden.
