# Contribuindo com o Calango

Estamos construindo a V1 em público. Leia [GOVERNANCE.md](GOVERNANCE.md), [AGENTS.md](AGENTS.md), [LICENSE](LICENSE), [CLA.md](CLA.md) e [TRADEMARKS.md](TRADEMARKS.md).

## Idioma

O idioma operacional é **português brasileiro**: issues, PRs, revisões, descrições de commits e documentação de trabalho. Nomes técnicos, símbolos de código, contratos de API e prefixos Conventional Commits permanecem estáveis. Textos da interface passam por packages/i18n (en, pt-BR, es); não traduzir contratos nem código por mera mudança editorial. Documentos legados serão traduzidos gradualmente. Contribuições em outros idiomas podem ser aceitas mediante tradução quando necessário.

## Como contribuir

1. Consulte [GitHub Issues](https://github.com/IAPro-Community/calango/issues), evite duplicatas e solicite atribuição de uma issue Ready.
2. Atualize develop e crie uma branch temporária no formato `<tipo>/<numero-issue>-<descricao>`: feat, fix, refactor, docs, test, ci, chore, spike, hotfix. Colaboradores externos usam forks; sessões paralelas usam worktrees isoladas.
3. Implemente somente o escopo, teste e abra draft PR vinculado para develop. ADR revisado para decisões arquiteturais.
4. Resolva conversas e aguarde o quality-gate. A revisão humana na develop é opcional. Mantenedor integra tarefas por squash na develop; branch temporária removida após merge. Promoção develop → main por PR com quality-gate, revisão humana independente após o último push e merge commit; preservar develop, sincronizar hotfixes da main. Nunca push direto nas branches permanentes.
5. Use Refs #N para entregas parciais e Closes #N somente quando o merge satisfaz todos os critérios; caso contrário encerre após validação.

Para promover uma versão à main, siga o [guia de publicação de releases](DEV/RELEASE.md).

O TODO V1 e roadmap antigos são referências históricas até migração completa. Não assumir atividades sem coordenação.

## Ambiente

Go 1.23+; Node.js 22.12+ para o workspace; Maestro conforme seus requisitos. Consulte package.json para comandos desktop.

```bash
go vet ./...
go test ./...
go run ./cmd/calango
```

Site: `docker build -t calango-site .` e `docker run --rm -p 8080:8080 calango-site`.

## Regras técnicas

Siga [DEV/CONVENTIONS.md](DEV/CONVENTIONS.md). Alterações de API começam em api/openapi.yaml. Migrações publicadas são imutáveis. Ações sensíveis exigem aprovação; nunca incluir segredos ou dados pessoais. Não copiar código de referências; respeitar ADR 0006 e exceções 0009/0011 e NOTICE.

## Pull requests

Um assunto por PR. Título Conventional Commits, por exemplo `feat(vault): implementar rotacao de chaves`. Informar motivo, escopo, critérios, comandos/resultados reais, riscos e limitações. Incluir testes de domínio e documentação quando aplicável. Aceitar CLA no primeiro PR conforme processo confirmado pelo mantenedor. Informar assistência de IA quando relevante. Na promoção para main, é proibido aprovar o próprio PR.

## Conduta e segurança

Respeite [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Vulnerabilidades nunca em issues públicas; siga [SECURITY.md](SECURITY.md).
