# Qualidade de engenharia

## Comandos na raiz

Requisitos: Go 1.23, Node 22.12+ e Git. Rode npm ci antes dos checks. Os linters têm versão explícita; primeira execução pode baixar ferramentas. Não usar latest nem alterar o lockfile inadvertidamente.

| Comando | Efeito |
|---|---|
| npm run format | Aplica gofmt e Biome nos fontes selecionados. |
| npm run format:check | Verifica formatação, sem modificar arquivos. |
| npm run lint | Go vet, Staticcheck e Biome. |
| npm run lint:fix | Correções seguras do Biome; revisar diff. |
| npm run contracts:check | Verifica tokens/i18n (inclusive tokens do tema Flutter), testes de geração e drift dos SDKs TypeScript e Dart. Não exige Dart instalado. |
| npm run sdk:generate:dart / npm run tokens:generate | Regenera o SDK Dart a partir do OpenAPI / os artefatos de tokens (CSS, TS, Dart e tema Flutter). |
| npm run typecheck | Compilação TypeScript do renderer e main/preload. |
| npm test | Testes Go, desktop main e componentes do renderer (Vitest). |
| npm run quality | Agrega os checks locais e build desktop. |

Biome cobre TS/TSX, JS, CSS e scripts selecionados; excluir SDK schema e trayMessages.ts gerados, dist e node_modules. Não editar artefatos gerados à mão. Não aplicar fixes inseguros automaticamente. Site HTML monolítico não está no escopo do Biome inicial; validar via smoke e issue própria se necessário. Pacotes Dart (packages/sdk-dart, packages/theme-flutter; ADR 0024): o job dart-packages do CI instala Flutter 3.47.9 do arquivo oficial com SHA-256 fixado e roda pub get --enforce-lockfile, dart format --output=none --set-exit-if-changed, dart/flutter analyze --fatal-infos e dart/flutter test. Arquivos gerados têm // dart format off. O app (apps/mobile, #67) entra nos mesmos gates. Go race é executado em runner Linux com CGO habilitado no workflow ci e, no workflow ci-nativa, também em Windows e macOS, junto com os testes do desktop e o smoke do daemon (ver CI_NATIVA.md).

## Testes do renderer

Issue #118. `npm run test:renderer -w apps/desktop` roda Vitest 5.1.0 com Testing Library e jsdom 28 (compatível com Node 22.12). Os testes ficam ao lado das telas (`apps/desktop/src/renderer/**/*.test.tsx`) e entram no `npm test` e no job `desktop` da CI, que alimenta o `quality-gate`.

- `src/renderer/test/fakes.ts` traz um `CalangoClient` falso tipado: todo método é `vi.fn` e rejeita se o teste não o definiu, então uma chamada inesperada à API falha. Também traz registros de exemplo e uma ponte `window.calango` controlável (`fakeBridge.emitState`, `quickConversation`).
- Cobertura: agentes e contas, chat (envio, reenvio com a mesma chave de idempotência, parar, aprovação dentro da conversa), aprovações, rotinas, consumo, configurações (cofre, abertura com o sistema, Maestro), auditoria, reconexão após reinício do daemon e troca de idioma (pt-BR, en, es). Cada tela tem carregamento, vazio e erro.
- Teclado e foco: navegação lateral por Tab com `aria-current`, envio e decisão de aprovação por Enter. Alertas com `role="alert"` onde a tela já os declara.
- `src/renderer/contrast.test.ts` calcula o contraste WCAG dos pares de tokens usados em `styles.css` (4,5:1 para texto, 3:1 para o contorno de foco).
- Sem snapshots: os testes procuram por papel, rótulo e texto traduzido.

## Adoção

Primeira rodada: aplicar format, revisar diff e corrigir diagnósticos de lint em commits separados. Não suprimir regras globalmente para forçar verde. CI deve falhar quando houver violações, sem escrever no código. Registrar resultados e revisar o baseline antes de tornar quality-gate obrigatório. Ferramentas fixadas por versão na configuração/commands; migrar para devDependencies com lockfile em ambiente conectado antes de estabilizar a cadeia. Não foram executadas ferramentas novas nesta sessão remota; não declarar CI verde sem evidência.

## Escopo

Preservar contratos e lógica em mudanças mecânicas. Novas dependências e regras precisam de revisão. Testes fake não equivalem a aceite real. Ver issue #51 e PR de governança #2.