# ADR 0020 — IAPro-SDK como biblioteca compartilhada

Status: proposto · issue #128 · primeiro recorte para revisão humana.

## Decisão

Consumir o módulo público `github.com/IAPro-Community/IAPro-SDK` na revisão imutável `v0.0.0-20261124222628-3f48e9b634a8`, com checksums obtidos pelo Go em CI do SDK (run 45500969304). MIT: licença e NOTICE preservados no repositório de origem; AGPL do Calango não muda. Revisão de procedência completa e gate jurídico ainda pendentes. Não copiar módulos entre produtos.

O SDK fornece a classificação de status HTTP e a decisão de avançar na cadeia explícita do agente. O gateway do Calango permanece autoridade de orçamento, auditoria, persistência e aprovação. Identidade de provider Codex é canônica no SDK. O SDK não decide credenciais, autorização, política comercial, storage ou execução. A cadeia já configurada é a autorização de fallback existente; nenhum provider externo à cadeia é selecionado. O CLI não é reexecutado após falha ambígua. Se houver possibilidade de ação de ferramenta ou CLI, o fallback só prossegue quando a falha ocorreu antes do dispatch (não instalado/signed out). Anthropic direto não executa ferramentas; preserva comportamento de fallback antes de texto emitido.

Não migrar autenticação, probes de quotas, handoff, coordenação de agentes, UI ou runtime de subprocesso sem evidência real e integração específica. O SDK não é um daemon obrigatório. Não alterar CODEX_HOME, MCP, flags de sandbox ou contratos OpenAPI nesta etapa. Não habilitar seleção automática baseada em quota ausente.

## Validação

Testes da cadeia e harness existentes, novo TestSDKChainSafety, testes Anthropic e quality-gate. CI do SDK e consumidores deve passar no SHA exato; revisão humana e integração conforme GOVERNANCE.md. Homologação real e paridade do épico permanecem abertas. Rollback: reverter PR e pin correspondente. Não declarar v0.1 estável nem encerrar #128 antes dos demais critérios.
