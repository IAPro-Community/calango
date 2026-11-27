# Governança do Calango v1.0

O Calango adota fluxo com branches permanentes main e develop, desenvolvimento orientado a issues e releases incrementais. Esta política entra em vigor após aprovação humana e integração do PR de governança; configurações do GitHub precisam de verificação independente.

## Idioma e responsabilidades

Português brasileiro é o idioma operacional: issues, PRs, revisões, commits e documentação de trabalho. Preservar nomes de APIs, símbolos de código, identificadores técnicos e catálogos de tradução. Documentação pública multilíngue continua disponível. Traduzir documentos legados gradualmente, sem alterar contratos ou histórico.

- Product Owner: visão, escopo, prioridades, critérios de aceite e releases.
- Mantenedores: políticas, revisão humana, integração e publicação.
- Liderança técnica: arquitetura, padrões e ADRs.
- Colaboradores: implementação, testes e documentação das issues.
- Revisores: revisão humana independente; ninguém aprova o próprio trabalho.
- Agentes de IA: execução delegada sob responsabilidade humana; sem aprovação, bypass ou publicação autônoma.

Qualquer colaborador pode propor mudanças. Alterações de escopo, licenciamento e política comercial exigem decisão dos responsáveis. Registrar decisões na issue e ADR quando aplicável. Não inferir responsáveis a partir de commits.

## Fluxo de trabalho

GitHub Issues registra escopo; GitHub Projects organiza prioridades e status; milestones representam entregas; especificações e ADRs definem contratos. Uma issue por unidade de trabalho, um responsável humano e uma atividade principal por sessão. Dependências explícitas e sub-issues para entregas independentes.

Branches temporárias a partir da develop, PR vinculado para develop, quality-gate aprovado, conversas resolvidas e squash merge. A revisão humana é opcional na develop. Promoção develop → main por PR com quality-gate aprovado, revisão humana independente após o último push e merge commit para preservar ancestralidade e evitar reapresentar commits; sincronizar correções da main de volta à develop por PR. Proibido push direto, force push, exclusão de main/develop e bypass permanente. Nunca excluir a develop após promoção. Exceções emergenciais exigem autorização, registro e restauração da política.

Ready: objetivo, escopo, critérios, dependências, prioridade e ausência de bloqueios. Done: evidência dos critérios, testes, CI, merge e validação pós-merge quando aplicável. A revisão humana independente é obrigatória para promover à main. Duplicatas são encerradas com motivo, sem contar como entregas.

## Registros retroativos

Trabalho anterior à governança recebe título [Retroativa], IDs legados, SHA da main auditada, commits originais, módulos, evidências registradas e limitações. Não recriar branches/PRs históricos nem inventar autoria, aprovação ou testes. Encerrar somente o recorte comprovado; entregas parciais permanecem abertas. Preservar autoria Git; validação de release é independente. Não reescrever a história.

## Congelamento da main

Após a integração inicial expressamente autorizada pelo PO, a main é congelada como referência de versão. Nenhuma alteração cotidiana ou promoção automática; apenas liberação explicitamente autorizada, mediante PR da develop e gates verificados. A develop permanece aberta à integração por PR. A configuração de proteção do GitHub deve ser comprovada separadamente; esta política não substitui ruleset.

## Releases e cadência

A main recebe mudanças somente por PR de release com origem na develop. Siga o [guia de publicação](DEV/RELEASE.md) para promoção, tag e verificação dos artefatos.

SemVer, tags v imutáveis, alpha/beta/rc quando necessário, commit rastreável da main, changelog, compatibilidade, limitações, instalação/atualização, artefatos e validação. Publicação autorizada por mantenedor. Triagem contínua, refinamento semanal quando útil, revisão de bloqueios, milestones e releases. Medir fluxo da equipe, não linhas de código individuais.

Consultar [CONTRIBUTING](CONTRIBUTING.md), [convenções](DEV/CONVENTIONS.md), [entrega técnica](DEV/TECHNICAL_DELIVERY.md) e [bootstrap](DEV/GOVERNANCE_BOOTSTRAP.md). Respeitar [SECURITY](SECURITY.md), [CLA](CLA.md) e código de conduta.
