# Módulo: desktop-electron

Fase 1 (base) · Fases 4–5 (ADE, navegador) · `apps/desktop` · ADR 0003

## Objetivo
App desktop para Windows, macOS e Linux. Casca em volta do `calango` local; também conecta a nós remotos.

## Estrutura
```
apps/desktop/
├── src/main/        processo principal: janela, bandeja, update, sidecar do calango
├── src/preload/     ponte segura (contextIsolation, sem nodeIntegration)
└── src/renderer/    React + TS + Vite
```

## Regras
- O main inicia `calango` (binário empacotado por plataforma), espera `/healthz`, reinicia com backoff se cair, mostra estado na bandeja.
- O renderer usa só `packages/sdk-ts`. Token do desktop vem pelo preload, nunca fica no `localStorage`.
- Seletor de nó: "Neste computador" e nós remotos pareados.
- Segurança Electron: `contextIsolation: true`, `sandbox: true`, CSP estrita, nada de `remote`.
- Atalho global para abrir a conversa rápida.
- Distribuição: NSIS assinado (Windows), DMG notarizado (macOS), `.deb` e AppImage (Linux). Auto-update assinado.

## Telas da fase 1
Barra lateral (DMs, agentes, rotinas, "Para você"), conversa com passos, cartão de aprovação, editor de agente, editor de rotina, uso e custos, configurações (você, agentes, modelos, conexões, cofre).

## Aceite
- Matar o processo `calango` → o app mostra "reiniciando" e volta sozinho em até 10 s.
- Instalador Windows não dispara aviso do SmartScreen por falta de assinatura.
