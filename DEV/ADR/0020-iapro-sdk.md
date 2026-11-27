# ADR 0020 — IAPro-SDK como biblioteca compartilhada

Status: proposto · issue #128 · primeiro recorte para revisão humana.

## Decisão

Consumir o módulo público `github.com/IAPro-Community/IAPro-SDK` na revisão imutável `v0.0.0-20261124230439-ece304ad4225`, com checksums obtidos pelo Go em CI do SDK (run 45506771570). MIT: licença e NOTICE preservados no repositório de origem; AGPL do Calango não muda. Revisão de procedência completa e gate jurídico ainda pendentes. Não copiar módulos entre produtos.

O SDK fornece a classificação de status HTTP e a decisão de avançar na cadeia explícita do agente. O gateway do Calango permanece autoridade de orçamento, auditoria, persistência e aprovação. Identidade de provider Codex é canônica no SDK. O SDK não decide credenciais, autorização, política comercial, storage ou execução. A cadeia já configurada é a autorização de fallback existente; nenhum provider externo à cadeia é selecionado. O CLI não é reexecutado após falha ambígua. Se houver possibilidade de ação de ferramenta ou CLI, o fallback só prossegue quando a falha ocorreu antes do dispatch (não instalado/signed out). Anthropic direto não executa ferramentas; preserva comportamento de fallback antes de texto emitido.

Não migrar autenticação, probes de quotas, handoff, coordenação de agentes, UI ou runtime de subprocesso sem evidência real e integração específica. O SDK não é um daemon obrigatório. Não alterar CODEX_HOME, MCP, flags de sandbox ou contratos OpenAPI nesta etapa. Não habilitar seleção automática baseada em quota ausente.

## Validação

Testes da cadeia e harness existentes, novo TestSDKChainSafety, testes Anthropic e quality-gate. CI do SDK e consumidores deve passar no SHA exato; revisão humana e integração conforme GOVERNANCE.md. Homologação real e paridade do épico permanecem abertas. Rollback: reverter PR e pin correspondente. Não declarar v0.1 estável nem encerrar #128 antes dos demais critérios.


## Revisão de segurança

O gateway aloca o identificador de uso antes do dispatch; falhas de ID, contabilização ou cálculo de custo bloqueiam o fallback via ErrLedger. O adapter declara MayHaveSideEffects; ausência da capacidade é tratada como efeitos possíveis. A decisão usa a evidência do adapter efetivamente resolvido, não o nome. Testes de cadeia verificam bloqueio de backup sem output quando há efeitos possíveis e falha de ID. Licença MIT integral e NOTICE do SDK acompanham a imagem de node; os futuros instaladores desktop devem incluí-los e verificar o conteúdo antes da publicação (#39, #40, #116).


## Segundo recorte — runtime Codex

Atualização do pin para `v0.0.0-20261125011527-b93285109b30`, com checksums e seis jobs aprovados no run SDK 45524394245. `providers/codex.RunExec` e `CheckLoginStatus` substituem a execução e decodificação local. O host mantém as flags existentes de sandbox/ephemeral, workspace, CODEX_HOME, configuração MCP, gateway, aprovações e orçamento. `--json` emite mensagens completas (não deltas de tokens). Contadores do terminal são usados se ambos medidos; ausência no sucesso mantém estimativa explícita; ausência em falha devolve Usage{} para que gateway contabilize a reserva. Sem replay de execução ambígua; sem token em argv, sem stderr bruto e sem herdar bearer anterior. Sessões continuam efêmeras. Probes, quota real, handoff, UI e persistência de coordenação não são ativados neste PR. Fixture local não comprova compatibilidade com CLI real (#53/#113). Rollback por reversão do PR/pin.
