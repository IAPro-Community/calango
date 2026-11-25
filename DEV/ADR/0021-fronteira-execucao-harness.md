# ADR 0021 — Fronteira de execução dos harnesses

Status: proposto, revisão humana necessária · Refs #113, #53, #54, #57, #117, #128.

## Matriz de garantias atuais

| Capacidade | Anthropic API direta | Codex CLI |
|---|---|---|
| Ferramentas MCP Calango | Não executa ferramentas | Gateway por turno; grants e aprovação de escrita |
| Comandos nativos | Não executa | CLI e sandbox próprio; não passam pelo gateway |
| Filesystem | Sem ferramenta local | workspace-write; leitura e isolamento dependem do CLI/OS |
| Rede | HTTP ao provedor | Política do CLI/OS; não é controlada pelo MCP |
| Orçamento | Reserva + max_tokens enviado | Reserva e ledger; limite rígido de saída não comprovado |
| Cancelamento | Contexto da requisição | Cancela processo principal; árvore de processos não comprovada |
| Credenciais | Chave de conta via vault | Login local e CODEX_HOME; ambiente deve ser explicitamente filtrado |

ReadOnlyHint é uma declaração do servidor MCP, não prova ausência de efeito. Conectores devem ser previamente confiáveis e autorizados. Ações externas de resultado ambíguo não podem ser repetidas automaticamente.

## Proposta de política

- Ambiente de CLI por allowlist; nenhuma chave Calango, token de outro turno, chave de API ou token Git herdado. Token MCP atual é injetado exclusivamente para o turno. Login por conta configurada. Não aceitar credencial implícita de ambiente como identidade de conta.
- O PR #135 deve preservar esta política quando migrar command factory ao SDK. Não criar segundo parser/runtime.
- Não prometer autorização de todos os comandos nativos. Até prova de isolamento, execução autônoma irrestrita não está homologada. Definir perfil somente leitura e perfil workspace explicitamente consentido; validar flags e limites contra versões reais dos providers antes de ativar.
- Não prometer limpeza de descendentes sem testes nativos. SDK deve ser autoridade de runtime comum; host Calango retém decisão de permissões, orçamento e auditoria.
- Não prometer confidencialidade contra processo malicioso rodando como o mesmo usuário; ambiente filtrado não impede leitura de arquivos, sockets, credenciais ou outras capacidades acessíveis ao usuário do processo.

## Gates restantes

Fixtures: ambiente, segredo, isolamento de contas, revogação, cancelamento antes/durante output, ausência de recibo, política de fallback. Homologação Windows do processo e descendentes e Codex/Anthropic real com orçamento explícito. Linux/macOS antes de anunciar suporte. Aprovação dos perfis, comunicação dos limites na UI e documentação pública. Sem fechamento de #113/#41 apenas por CI verde.
