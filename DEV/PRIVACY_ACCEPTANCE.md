# Privacidade: escopo e critérios de aceite (#46)

Status: proposta para revisão do PO; não representa implementação de exportação ou eliminação integral.

- Exportação autenticada de agentes, configurações não secretas, memórias, conversas, rotinas, decisões, uso e metadados pertencentes ao titular. Formato JSON versionado, manifesto e limites de tamanho. Excluir tokens, chaves, envelopes criptográficos, credenciais MCP e conteúdo de outros titulares.
- Eliminação: inventariar dependências, cancelar execuções, impedir novos trabalhos durante a operação, revogar sessões e validar exclusão transacional dos registros permitidos. Confirmar escopo de dispositivos, contas e conectores antes de remover. Não disponibilizar exclusão destrutiva enquanto retenção de auditoria e backups não estiver definida.
- Auditoria append-only: decidir quais metadados mínimos devem ser retidos, justificativa, prazo e procedimento de pseudonimização que preserve integridade verificável. Não alterar eventos existentes silenciosamente.
- Backups e logs: definir prazo, expiração, restauração sem reintroduzir dados eliminados e responsabilidade operacional. A exclusão local não implica apagar dados enviados a providers externos.
- Primeiro uso: detectar node, modelos, harness, vault e conectores localmente; exibir somente disponibilidade e metadados autorizados, nunca segredos, prompts, resultados ou ambiente bruto. Não enviar telemetria sem consentimento.
- Testes de aceitação: isolamento entre titulares e escopos, exportação sem segredos, eliminação idempotente, falha parcial e recuperação, revogação de tokens, restauração de backup, ausência de cache e inspeção de payloads/logs com fixtures sintéticas.

Entrega parcial deste PR: Cache-Control no-store em respostas autenticadas (inclusive falhas) e pareamento. O header não é garantia contra cópia pelo cliente. A exportação, eliminação integral e descoberta inicial continuam abertas e exigem aprovação de escopo e implementação.
