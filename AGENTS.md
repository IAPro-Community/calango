# AGENTS.md — Calango

Assistente pessoal com core Go, desktop Electron, mobile Flutter e execução em PC/VPS.

## Antes de codificar

1. Leia GOVERNANCE.md e CONTRIBUTING.md.
2. Leia issue atribuída, critérios, dependências e PRs.
3. Consulte DEV/INDEX.md, arquitetura, ADRs e especificações do módulo relevante.
4. Confirme Ready e responsável humano. Use branch da issue e worktree isolada por sessão. Não carregue todo DEV/ nem assuma trabalho alheio.

## Regras obrigatórias

- Código original: agentes.chat é referência somente de leitura; não copiar código, textos, testes ou assets. ADR 0006.
- Maestro: consumir pacote npm oficial (ADR 0009), sem copiar repositório; manter NOTICE.
- AgenticOS: somente módulos autorizados no ADR 0011, reescritos em Go/TS/Dart com atribuição; preservar NOTICE e third_party/agenticos/.
- Binário calango em cmd/calango; clientes não executam agentes, acessam a API.
- Chamadas de modelo passam pelo gateway com orçamento e registro de custo.
- Ações sensíveis exigem aprovação; segredos nunca vão ao cliente ou logs.
- Migrações imutáveis; novas decisões arquiteturais exigem ADR.
- Commit/push/PR somente com autorização delegada na branch da issue. Nunca push direto na main, autoaprovação, bypass, merge ou publicação autônoma.
- AGPL-3.0, CLA, verificação de licenças antes de adicionar dependências.
- Português brasileiro em issues, PRs, revisões, commits e documentação operacional; preservar identificadores e contratos técnicos. Interface por packages/i18n.

## Comandos

```bash
go vet ./...
go test ./...
go run ./cmd/calango
npm run typecheck -w apps/desktop
npm run build -w apps/desktop
npm run test:main -w apps/desktop
```

## Após trabalhar

Registre escopo, comandos, resultados reais, limitações e próximos passos na issue/PR. Atualize ADR/especificações quando afetados. Não atualize rotineiramente ACTIVE/HANDOFF/WORKLOG/VERIFY globais ou checkboxes do roadmap; preservar legados durante transição. Solicite revisão humana. Não encerre issue antes do aceite.
