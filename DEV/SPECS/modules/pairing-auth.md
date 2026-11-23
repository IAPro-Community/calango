# Módulo: pairing-auth

Fase 1 (token local) · Fase 2 (pareamento) · Pacote `internal/auth` · Tabelas `devices`, `pairing_codes`

## Objetivo
Só dispositivos autorizados falam com um nó.

## Fluxo de pareamento
1. No desktop (ou no log da VPS): "Parear dispositivo" gera código de 8 caracteres + QR, válido 5 min, uso único. Guardado como hash.
2. O app envia o código para `POST /v1/pair` pelo transporte.
3. O nó devolve um token de dispositivo (aleatório, 256 bits), guardado no keychain do celular. No nó, só o hash.
4. Lista de dispositivos com último acesso; revogar invalida na hora.

## Regras
- Toda rota exige `Authorization: Bearer <token>`, exceto `/healthz` e `/v1/pair`.
- `/v1/pair` tem limite de tentativas (5 por minuto por origem).
- Escopos de token: `full`, `approve_only` (para um relógio, por exemplo), `read`.
- Na primeira subida do nó VPS, senha de administrador obrigatória; sem credencial padrão.

## Aceite
- Código usado duas vezes falha na segunda.
- Token revogado recebe `unauthorized` na próxima chamada.
