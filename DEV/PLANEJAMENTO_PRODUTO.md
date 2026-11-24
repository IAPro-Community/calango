# Planejamento operacional do Calango

Base reconciliada: [a623dd17](https://github.com/IAPro-Community/calango/commit/a623dd17cb86f5f3efffa9664c6f56d9ee308114). GitHub Issues é a fonte das atividades; este documento é uma referência inicial, não um segundo backlog.

## Histórico

26 registros canônicos encerrados e atribuídos a @FernandoBolzan conforme reconhecimento do PO; #15/#16 são duplicatas excluídas. 11 tarefas de continuação preservam critérios não comprovados: #9 → #53; #11 → #54; #13 → #55; #17 → #56; #18 → #57; #20 → #58; #21 → #59; #22 → #60; #23 → #61; #27 → #62; #28 → #63. Encerrar recorte histórico não significa aceitar funcionalidade integral. CODEX_HOME estável entregue em e8d3de3; #59 cobre validação real e política de migração.

## Épicos e entregas

| Épico | Marco proposto | Capacidade | Issues vinculadas | Gate |
|---|---|---|---|---|
| #31 | v0.2.0 | Assistente local | 39, 40, 41, 53–62, 110–112 | Aceite #41 |
| #32 | v0.3.0 | Celular | 64–71 | Aceite #71; transporte #43 |
| #33 | v0.4.0 | Híbrido | 72–81 | Aceite #81 |
| #34 | v0.5.0 | ADE | 82–90 | Aceite #90; preview #87 depende #91 |
| #35 | v0.6.0 | Browser | 91–98 | Aceite #98 |
| #36 | v0.7.0 | Colaboração | 99–103 | Aceite #103 |
| #37 | Transversal / v1.0.0 | Qualidade e lançamento | 42–52, 63 | Aplicável a cada fase |
| #38 | Pós-V1 | Expansão | 104–109 | Fora do escopo V1 |

82 issues de planejamento (#31–#112), incluindo 8 épicos e 74 unidades de trabalho; as referências são vínculos textuais, não sub-issues nativas nem milestones já configurados.

## Ordem e dependências

- Priorizar #41 (aceite local): #39, #53, #54, #55, #57, #59, #60, #62; #51 verifica CI. #40 e harnesses #110–#112 são entregas V1 posteriores conforme capacidade. #56, #58, #61 e #63 devem ser refinadas por risco e fase.
- Decisões #42 e #43 são do produto/proprietários; #44 define domínio/preços/copy. Não executar ações reais de modelos ou ações externas sem autorização.
- #64 depende #58; #65 depende #43; #67 depende #64–#66; #69 depende #67/#68; #70 e #71 depois dos componentes móveis.
- #72 depende #45; #73 depende #65; #74 depende #72/#73; #76 depende #75/#80; #77 depende #75/#76; #78 depende #73; #79 depende #59/#72.
- #83 depende #82; #84/#85/#86 integram o workspace; #87 depende #91 (fundação antecipável sem ciclo); #89 depende #84; #90 depende #74 e tarefas ADE.
- #92/#93 dependem #91; #94 depende #72/#91; #96 depende #65/#67/#77/#95; #98 valida #92.
- #100/#101 dependem #99; #102 depende #76/#77/#101; #103 valida #99–#102. Pós-V1 permanece separado.

## Distribuição e rotina

Todas as novas atividades estão no backlog, sem atribuição automática a Fernando. PO define responsáveis e prioridades; refinamento subdivide tarefas grandes; dependências e bloqueios explícitos. Cada responsável assume uma issue principal por vez; branch vinculada, PR, revisão humana e CI. Não atribuir trabalho sem decisão da equipe. Acompanhamento por GitHub Issues até criação do Project; após configuração, mover os campos e visões para Calango Product. Reunião de refinamento semanal quando útil, revisão contínua de bloqueios, evidências e gates de release.

## Pendências administrativas

PR #2 draft, sincronização com main e CI ainda necessárias; não integrar sem revisão humana. Criar GitHub Project, labels, milestones reais, relacionamentos nativos e ruleset conforme DEV/GOVERNANCE_BOOTSTRAP.md. Não apagar TODO/ROADMAP sem revisão de cobertura; o histórico permanece preservado. Não publicar release automaticamente.
