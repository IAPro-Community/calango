# Privacidade: escopo e critérios de aceite (#46)

Status: proposto para revisão do PO; não representa exportação nem exclusão completas.

- Exportação autenticada de agentes, configurações não secretas, memórias, conversas, rotinas, decisões, uso e metadados do titular. Formato JSON versionado, manifesto e limites de tamanho. Excluir tokens, chaves, envelopes cifrados, credenciais de MCP e conteúdo de outros titulares.
- Exclusão: inventariar dependências, cancelar execuções, impedir trabalho novo durante a operação, revogar sessões e validar a remoção transacional dos registros permitidos. Confirmar o escopo de dispositivos, contas e conectores antes de removê-los. Não expor exclusão destrutiva antes de definir retenção de auditoria e backups.
- Auditoria append-only: decidir quais metadados mínimos devem ser retidos, a justificativa, o período de retenção e o procedimento de pseudonimização que preserve a integridade verificável. Não alterar eventos existentes silenciosamente.
- Backups e logs: definir retenção, expiração, restauração sem reintroduzir dados excluídos e responsabilidade operacional. A exclusão local não implica exclusão de dados enviados a provedores externos.
- Primeiro uso: detectar node, modelos, harness, vault e conectores localmente; mostrar apenas disponibilidade e metadados autorizados, nunca segredos, prompts, resultados ou ambiente bruto. Não enviar telemetria sem consentimento.
- Testes de aceite: isolamento entre titulares e escopos, exportação sem segredos, exclusão idempotente, falha parcial e recuperação, revogação de token, restauração de backup, ausência de cache e inspeção de payloads/logs com fixtures sintéticas.

Entrega parcial neste PR: `Cache-Control: no-store` em respostas autenticadas (inclusive falhas) e pareamento. O cabeçalho não garante que o cliente não possa copiar dados. Exportação, exclusão completa e descoberta inicial permanecem abertas e exigem aprovação de escopo e implementação.