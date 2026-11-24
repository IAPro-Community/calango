# Revisão técnica e de produto — 23/11/2026

Base main a623dd17. Revisão estática de código, commits, configuração, documentação e issues; não é certificação, auditoria integral de segurança ou nova execução de testes.

## Direção

A arquitetura de um daemon Go que executa agentes em PC/VPS, clientes leves e contrato OpenAPI compartilhado é coerente com a visão. SQLite local, gateway com reserva de orçamento, aprovações, MCP por turno, rotina persistida e integração Maestro são evidências de fundação relevante. Electron já existe. Não introduzir segundo motor de agentes, reescrita de stack ou Flutter completo antes de provar o fluxo local. ADE/browser/equipes são expansão, não evidência de maturidade do MVP.

A hipótese de diferenciação é controle do usuário sobre execução, política, custos e evidências entre nós, não somente PC+VPS+celular.

## Achados e ações

| Prioridade | Achado | Ação |
|---|---|---|
| P1 | Codex hoje descarta stdout/stderr e entrega resposta final com tokens estimados; não é streaming real nem custo exato. | #53, #60 |
| P1 | MCP controla suas próprias ferramentas; workspace-write não implica aprovação de todo comando do CLI. Limites efetivos e processos descendentes precisam de prova. | #113, #54, #57 |
| P1 | Desktop Electron expõe token local ao renderer via preload conforme ADR 0019; política geral de que segredos nunca chegam ao cliente é ampla demais. Diferenciar credencial de sessão do vault e revisar escopo/rotação e modelo de confiança. | #55, #113 |
| P1 | CI da main não inclui lint TS, formatação TS, build/test desktop, Staticcheck e SDK drift explícito. | #51, este PR |
| P1 | Windows-first ainda sem instalação assinada e fluxo real completo; registros de testes fake/Linux não atendem gate. | #39, #41, #55, #62 |
| P1 | Home estável Maestro entregue em e8d3de3; sessão real e migração não comprovadas. | #59 |
| P1 | Tailscale comercial não decidido; PLANO.md afirma transporte decidido. | #43 |
| P2 | SDK tem schema gerado, mas cliente HTTP index.ts manual. Verificar tipos de caminhos/métodos e testes de contrato; geração não garante SDK inteiro. | #51, #66 |
| P2 | PLANO.md ainda afirma fase 0 apenas; ROADMAP ainda descreve home versionada; documentação de entrega contém evidências antigas. | #1, #47, #59 |
| P2 | Desktop usa polling e poucas evidências de UI automatizada; websocket com header não é diretamente acessível ao browser. Não mudar contrato sem ADR. | #41, #54 |
| P2 | Site promove capacidades planejadas como se disponíveis. | #44, #49 |

## Sequência recomendada

1. Fechar governança e baseline de CI (#1/#51).
2. Provar conversa Codex/Anthropic, custo e aprovações com autorização (#53/#54/#60/#113).
3. Validar Maestro, keyring, bandeja, backup e instalador Windows (#39/#55/#59/#62/#63).
4. Aceite funcional #41 e primeira versão local limitada, com escopo explícito.
5. Decidir transporte #43, seguir F2/F3; depois ADE/browser e colaboração conforme gates.

Não considerar todo backlog V1 como condição para primeira alpha local. Releases públicas exigem verificação de instalação, atualização, segurança e limitações. Testes com fakes devem continuar na CI sem credenciais.

## Engenharia

Este PR adiciona configuração de formatter/linter, scripts locais, validação de contratos e CI desktop/Go. O baseline ainda precisa ser executado e corrigido; não há resultado positivo comprovado. Não reformatar migrations nem arquivos gerados. Ver QUALIDADE.md. Não aplicar correções unsafe em massa.

## Governança

PR #2 ainda draft; main avançou depois da criação e precisa de sincronização/revisão. Nenhum merge automático. As issues históricas de Fernando foram encerradas por recorte; as 11 continuações permanecem abertas. Épicos e dependências registrados em issues e no planejamento da branch de governança; GitHub Project, labels e milestones administrativos ainda pendentes. A nova issue #113 deve integrar o gate #41.
