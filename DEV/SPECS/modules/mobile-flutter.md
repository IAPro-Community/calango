# Módulo: mobile-flutter

Fase 2 · `apps/mobile` · ADR 0004

## Objetivo
Controle remoto do Calango no celular: ver, aprovar, conversar, agendar.

## Telas
1. Boas-vindas + parear (câmera para QR ou código).
2. Nós: PC e VPS, online/offline, desde quando, o que espera por cada um.
3. "Para você": aprovações, perguntas, entregas.
4. Conversas (DMs) com passos resumidos.
5. Cartão de aprovação com ação, alvo, argumentos, aprovar/recusar/aprovar sempre.
6. Rotinas: lista, rodar agora, pausar.
7. Tarefas (Quadro simplificado).
8. Configurações: dispositivos, plano, notificações.
9. Controle de tela do PC (fase 5, pago).

## Regras
- Cliente só via `packages/sdk-dart`. Tema só via tokens gerados.
- Token no Keychain (iOS) / Keystore (Android).
- Push abre direto no item (aprovação, mensagem).
- Funciona com mais de um nó; seletor no topo.
- Compra pelo sistema da loja; plano reconhecido em todos os clientes.

## Aceite
- Aprovar uma ação do PC estando em 4G, via Tailscale, em menos de 3 toques a partir do push.
