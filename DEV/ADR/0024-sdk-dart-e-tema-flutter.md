# ADR 0024 — SDK Dart e tema Flutter gerados por scripts Node do repositório

Status: proposto · issue #66 (F2-T03) · fundação da fase 2 antecipada, antes do aceite da fase 1, para destravar #67.

## Contexto

O ADR 0004 decidiu que `api/openapi.yaml` gera `packages/sdk-dart` e que `packages/tokens/tokens.json` gera o `ThemeData`, sem cliente escrito à mão. O CI precisa detectar drift, e a geração tem de ser determinística. A máquina de desenvolvimento e o job de contratos não têm Java; o `npm run contracts:check` roda hoje só com Node.

## Alternativas avaliadas

- **openapi-generator (`dart`, `dart-dio`)**: maduro, mas exige Java (o wrapper npm baixa um jar); `dart-dio` ainda depende de `build_runner`. Rejeitado por trazer JVM ao fluxo local e ao CI.
- **swagger_parser (pub.dev, MIT)**: Dart puro, mas gera código para `retrofit`/`dio` + `json_serializable`/`freezed`, o que exige `build_runner` e um conjunto grande de dependências de geração e de runtime. A checagem de drift passaria a exigir Dart instalado em todo `contracts:check`, e o suporte a OpenAPI 3.1 (`type: [x, "null"]`) é parcial.
- **Gerador Node do próprio repositório** (escolhido): segue o padrão de `generate-tokens.mjs`, `generate-i18n.mjs` e dos assets. Sem Java, sem `build_runner`, sem dependência de runtime além de `package:http`.

## Decisão

- `scripts/generate-sdk-dart.mjs` lê o OpenAPI com `js-yaml` 4.3.2 (MIT, fixado em `devDependencies`) e escreve `packages/sdk-dart/lib/src/generated/{models,client}.dart`: classes com `fromJson`/`toJson`, enums com o valor de fio, e um método do `CalangoClient` por operação REST. Só o transporte HTTP (`lib/src/transport.dart`) é escrito à mão; ele usa o envelope de erro gerado.
- O gerador aceita apenas o subconjunto de OpenAPI que o contrato usa (objetos, `allOf`, `$ref` locais, enums de string, `[tipo, "null"]`, parâmetros de path/query/header, JSON e respostas texto) e falha com o caminho do schema em qualquer outra construção (`oneOf`, `anyOf`, `discriminator`, tipos não suportados, nomes que viram identificadores inválidos ou colidem). Nenhuma construção nova vira código silenciosamente errado: quem mudar o contrato estende o gerador no mesmo PR.
- Rotas de WebSocket (resposta só `101`, hoje `GET /v1/events`) ficam fora do cliente REST e são listadas no comentário da classe gerada.
- `scripts/generate-tokens.mjs` passa a escrever também `packages/theme-flutter/lib/src/generated/tokens.dart` com constantes tipadas (`Color`, raios, `BoxShadow`, `Cubic`, `Duration`). `packages/theme-flutter/lib/calango_theme.dart` só mapeia esses tokens para papéis do Material (`ColorScheme` claro e escuro, tipografia, formas); não declara cores avulsas. Sombras `inset` do CSS não têm equivalente no Flutter e são omitidas, com nota no arquivo gerado.
- Os arquivos gerados começam com `// dart format off`: a saída é estável sem depender da versão do formatter. `npm run contracts:check` compara byte a byte (mesma entrada, mesma saída) e não exige Dart.
- Toolchain fixada: Flutter 3.47.9 (Dart 3.13.7), baixado do arquivo oficial com SHA-256 verificado no job `dart-packages` do CI. `pubspec.lock` dos dois pacotes é versionado e o CI usa `--enforce-lockfile`.

## Consequências

- Manter o gerador é custo do projeto; em troca, o contrato e o código gerado ficam legíveis e revisáveis no diff, e o drift check roda em qualquer máquina com Node.
- Enums rejeitam valores desconhecidos (`FormatException`). Um nó mais novo que o app pode quebrar a decodificação até o app ser regenerado; revisar quando houver versionamento de API.
- Espaçamentos não existem em `tokens.json`; o tema não inventa valores. As fontes (Bricolage Grotesque, Instrument Sans, JetBrains Mono) são referenciadas pelo nome e devem ser empacotadas pelo app (#67).
- Rollback: reverter o PR; nenhum contrato da API muda.
