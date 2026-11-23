# ADR 0003 — Desktop em Electron com `calango` como processo filho

Status: aceita · 2026-11-23 (substitui a proposta inicial de Wails)

## Contexto
O desktop precisa de navegador real embutido (abas que o agente dirige e a pessoa assume), editor e terminal para a ADE, bandeja, auto-update e instalador para Windows, macOS e Linux.

## Decisão
Electron. O processo principal inicia o binário `calango` como processo filho, vigia sua saúde e o reinicia se cair. O renderer (React + TypeScript + Vite) fala com o `calango` pela API local, igual a qualquer outro cliente. Navegador interno com `WebContentsView`; o agente dirige pelo protocolo de depuração do Electron.

## Alternativas
- Wails (Go + WebView do sistema): menor, mas WebView do sistema não dá abas Chromium completas nem controle por CDP uniforme entre plataformas.
- Tauri: mesma limitação de WebView.

## Consequências
- Instalador maior (~100 MB). Aceito.
- Instaladores assinados desde o primeiro release público (Windows code signing, macOS notarização).
- O `calango` precisa ser empacotado por plataforma junto com o app.
