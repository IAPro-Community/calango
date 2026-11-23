# Convenções de código

## Geral

- Idioma do código, identificadores e comentários: **inglês**. Idioma dos docs `DEV/` e da interface padrão: **português**, com inglês e espanhol desde o início.
- Toda string de interface passa pelo catálogo de traduções (`packages/i18n`). Nada de texto fixo em tela.
- Commits pequenos, mensagem no imperativo. Nunca commitar segredo (`.env` fica fora do git).

## Go (`cmd/`, `internal/`)

- Go 1.22+. `gofmt` e `go vet` limpos. `staticcheck` quando o CI existir.
- `internal/core` é domínio puro: sem rede, sem disco, sem relógio global (relógio injetado).
- Erros com contexto (`fmt.Errorf("routine %s: %w", id, err)`); erros sentinela exportados para o que o chamador precisa distinguir.
- `context.Context` como primeiro parâmetro em tudo que faz I/O.
- Sem pânico em código de produção, exceto invariantes impossíveis.
- Dependências externas só com motivo escrito no PR. Preferir biblioteca padrão.
- Interfaces declaradas no pacote que as consome, não no que as implementa.

## Testes

- Teste de unidade obrigatório para regra de domínio.
- Testes não dependem de rede, de Docker nem de chaves reais. Modelo de IA é falso nos testes (provider fake).
- Banco nos testes: SQLite em memória (`file::memory:?cache=shared`).
- Nome: `TestUnidade_Comportamento`.

## TypeScript (desktop, web)

- `strict: true`, sem `any` (usar `unknown` com guarda).
- React com componentes em PascalCase, hooks com prefixo `use`.
- Cliente da API: só o gerado (`packages/sdk-ts`). Nunca `fetch` à mão para a API do `calango`.
- Estilo: tokens de `packages/tokens`; nada de cor solta no CSS.

## Dart (mobile)

- `flutter analyze` limpo, `dart format`.
- Cliente da API: só o gerado (`packages/sdk-dart`).
- Tema: `ThemeData` gerado de `packages/tokens`.

## API

- Contrato primeiro: mudar `api/openapi.yaml`, regenerar SDKs, depois implementar.
- Rotas versionadas: `/v1/...`. Erros em JSON `{ "error": { "code": "...", "message": "..." } }`.
- Códigos de erro são palavras de uma lista fixa (`budget_exceeded`, `approval_required`, `not_found`, `invalid_input`, `unauthorized`, `node_offline`).
- Tempo real por WebSocket em `/v1/events` com eventos tipados (ver `DATA_MODEL.md`).

## Banco

- Migrações numeradas em `internal/store/migrations/NNNN_nome.sql`, imutáveis depois de commitadas.
- Toda tabela de domínio tem `id` (ULID texto), `created_at`, `updated_at` e `owner_id` quando aplicável.
