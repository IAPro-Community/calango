# Auditoria retroativa da main — 23/11/2026

Base auditada: [1e8868b](https://github.com/IAPro-Community/calango/commit/1e8868ba0930792cc949d7d1b06ec5bc5eca9fbd). Nove commits históricos localizados; três após dee8efa: a3f04ae (core), ff11449 (desktop), 1e8868b (documentação). Confronto entre histórico, arquivos alterados, TODO, roadmap e VERIFY. Evidências de testes são registros do repositório, não testes executados nesta auditoria. Metadados de autoria não retornados pelo conector; não atribuir autoria ou inventar revisão. Não reescrever histórico.

## Matriz de rastreabilidade

| Issue | IDs originais | Entrega | Classificação |
|---|---|---|---|
| [#3](https://github.com/IAPro-Community/calango/issues/3) | F0-T01–T05 | Fundação | Histórico |
| [#4](https://github.com/IAPro-Community/calango/issues/4) | F0-T04/F0-T06/Q05 | CI/OpenAPI/build | Histórico |
| [#5](https://github.com/IAPro-Community/calango/issues/5) | F0-T07/F1-T22 | Tokens | Histórico |
| [#6](https://github.com/IAPro-Community/calango/issues/6) | Q26 | Site | Histórico |
| [#7](https://github.com/IAPro-Community/calango/issues/7) | F1-T01/T02 | SQLite | Histórico |
| [#8](https://github.com/IAPro-Community/calango/issues/8) | F1-T03/T06 | Gateway/agentes | Histórico |
| [#9](https://github.com/IAPro-Community/calango/issues/9) | F1-T04/T05/Q32 | Provedores | Parcial |
| [#10](https://github.com/IAPro-Community/calango/issues/10) | F1-T07/Q22 | Conversas | Histórico |
| [#11](https://github.com/IAPro-Community/calango/issues/11) | F1-T08/Q15 | Aprovações | Parcial |
| [#12](https://github.com/IAPro-Community/calango/issues/12) | F1-T09/Q16/Q33 | Rotinas | Histórico |
| [#13](https://github.com/IAPro-Community/calango/issues/13) | F1-T10/Q12/Q13/Q28/Q29/Q31 | Vault | Parcial |
| [#14](https://github.com/IAPro-Community/calango/issues/14) | F1-T14/Q14/Q30 | Token | Histórico |
| [#17](https://github.com/IAPro-Community/calango/issues/17) | F1-T11/Q17 | Memória | Parcial |
| [#18](https://github.com/IAPro-Community/calango/issues/18) | F1-T12/Q20 | MCP | Parcial |
| [#19](https://github.com/IAPro-Community/calango/issues/19) | F1-T13/Q23 | REST/WebSocket | Histórico |
| [#20](https://github.com/IAPro-Community/calango/issues/20) | F1-T15/Q19 | Pareamento | Parcial |
| [#21](https://github.com/IAPro-Community/calango/issues/21) | F1-T16/Q24 | Maestro | Parcial |
| [#22](https://github.com/IAPro-Community/calango/issues/22) | F1-T17/Q18 | Custos | Parcial |
| [#23](https://github.com/IAPro-Community/calango/issues/23) | F1-T18/Q07–Q11 | Auditoria | Parcial |
| [#24](https://github.com/IAPro-Community/calango/issues/24) | F1-T20/T21 | Electron | Histórico |
| [#25](https://github.com/IAPro-Community/calango/issues/25) | F1-T22/T26 | SDK/i18n | Histórico |
| [#26](https://github.com/IAPro-Community/calango/issues/26) | F1-T23/Q21 | Telas | Histórico |
| [#27](https://github.com/IAPro-Community/calango/issues/27) | F1-T24 | Tray/autostart | Parcial |
| [#28](https://github.com/IAPro-Community/calango/issues/28) | Transversal | Backup | Parcial |
| [#29](https://github.com/IAPro-Community/calango/issues/29) | Transversal | Segredos/telemetria | Histórico |
| [#30](https://github.com/IAPro-Community/calango/issues/30) | D1/D1a/D2/D3/D4/D6/Q01/Q02/Q11 | Decisões | Histórico |

26 registros canônicos; 15 históricos encerrados e 11 parciais abertos. #15 e #16 encerradas como duplicatas de #13 e #14 respectivamente; excluir de métricas de entrega. O encerramento histórico não implica aceite da V1, CI atual verde ou aprovação humana retroativa. Issues parciais incluem gaps que devem ser refinados sem reimplementar o entregue.

## Próximos passos

- Confirmar CI do commit atual, execução real de modelos, ações externas, Windows/macOS, instalação assinada e critérios de saída da fase 1.
- Resolver divergência CODEX_HOME e falha de atualização Maestro.
- Pendências sem implementação (fases 2–6, empacotamento, decisões legais e transporte, entre outras) não foram importadas nesta rodada.
- Refinar cada pendência por unidade de trabalho e mapear cobertura de todos os IDs antes de arquivar TODO/roadmap legados.
- Não alterar autoria Git. Revisar atribuições com equipe; responsáveis futuros não equivalem a autores históricos.
- Nenhum merge da governança autorizado; sincronizar branch e verificar CI antes de revisão humana.
