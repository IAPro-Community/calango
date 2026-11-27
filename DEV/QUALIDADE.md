# Qualidade de engenharia

## Comandos na raiz

Requisitos: Go 1.23, Node 22.12+ e Git. Rode npm ci antes dos checks. Os linters têm versão explícita; primeira execução pode baixar ferramentas. Não usar latest nem alterar o lockfile inadvertidamente.

| Comando | Efeito |
|---|---|
| npm run format | Aplica gofmt e Biome nos fontes selecionados. |
| npm run format:check | Verifica formatação, sem modificar arquivos. |
| npm run lint | Go vet, Staticcheck e Biome. |
| npm run lint:fix | Correções seguras do Biome; revisar diff. |
| npm run contracts:check | Verifica tokens/i18n, testes de geração e drift do SDK. |
| npm run typecheck | Compilação TypeScript do renderer e main/preload. |
| npm test | Testes Go e desktop main. |
| npm run quality | Agrega os checks locais e build desktop. |

Biome cobre TS/TSX, JS, CSS e scripts selecionados; excluir SDK schema e trayMessages.ts gerados, dist e node_modules. Não editar artefatos gerados à mão. Não aplicar fixes inseguros automaticamente. Site HTML monolítico não está no escopo do Biome inicial; validar via smoke e issue própria se necessário. Flutter ainda não existe: dart format --output=none --set-exit-if-changed, flutter analyze e flutter test passam a ser gates ao introduzir o app. Go race é executado em runner Linux com CGO habilitado no workflow ci e, no workflow ci-nativa, também em Windows e macOS, junto com os testes do desktop e o smoke do daemon (ver CI_NATIVA.md).

## Adoção

Primeira rodada: aplicar format, revisar diff e corrigir diagnósticos de lint em commits separados. Não suprimir regras globalmente para forçar verde. CI deve falhar quando houver violações, sem escrever no código. Registrar resultados e revisar o baseline antes de tornar quality-gate obrigatório. Ferramentas fixadas por versão na configuração/commands; migrar para devDependencies com lockfile em ambiente conectado antes de estabilizar a cadeia. Não foram executadas ferramentas novas nesta sessão remota; não declarar CI verde sem evidência.

## Escopo

Preservar contratos e lógica em mudanças mecânicas. Novas dependências e regras precisam de revisão. Testes fake não equivalem a aceite real. Ver issue #51 e PR de governança #2.