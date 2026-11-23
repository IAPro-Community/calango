# Módulo: computer-use

Fase 5 · Pacote `internal/computer` (PC apenas)

## Objetivo
O agente opera apps de desktop; a pessoa controla a tela do PC pelo celular.

## Regras
- **Desligado por padrão.** Liga por app, no próprio PC (nunca remotamente).
- Ações: captura de tela, clique, teclado, rolagem, listar janelas.
- Enquanto ativo: indicador visível no PC e botão de encerrar sempre à mostra.
- Controle de tela pelo celular (plano pago): stream de vídeo pelo transporte + entrada remota; precisa estar ligado no PC; auditoria de início e fim.
- macOS exige permissões de Acessibilidade e Gravação de Tela; o app guia a pessoa.
- Apps sensíveis (banco, gerenciador de senhas) bloqueados por padrão.

## Aceite
- Com o recurso desligado, nenhuma ferramenta de computador aparece para o agente.
- Encerrar no PC derruba a sessão remota em menos de 1 s.
