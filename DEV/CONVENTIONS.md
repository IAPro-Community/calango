# Convenções de engenharia

## Geral

- Português brasileiro em issues, PRs, revisões, descrições de commits e documentação operacional. Identificadores, APIs, nomes de arquivos e símbolos técnicos permanecem estáveis.
- Interface em en (padrão), pt-BR e es via packages/i18n; sem textos fixos na tela. Ordem: escolha explícita > preferência salva > idioma do navegador/SO > en.
- Branch por issue; PR e commit squash com Conventional Commits: type(scope): descrição; indicar ! e BREAKING CHANGE quando incompatível. Escopos seguem módulos (core, api, store, config, agents, gateway, harness, memory, routines, approvals, mcp, maestro, auth, vault, audit, pairing, desktop, mobile, web, cloud, ade, browser, terminal, teams, billing, licensing, ci, deploy, release, deps).
- Commits pequenos e imperativos; nunca segredos (.env fora do Git).
- Revisão humana e CI obrigatórias antes do merge.

## Go

- Go 1.25+; gofmt e go vet sem problemas; Staticcheck quando incorporado à CI.
- internal/core é domínio puro: sem rede, disco ou relógio global; injetar relógio.
- Erros com contexto (fmt.Errorf com %w), sentinelas exportadas para casos distinguíveis.
- context.Context primeiro parâmetro em operações de I/O; sem panic em produção salvo invariantes impossíveis.
- Justificar dependências no PR; preferir biblioteca padrão; interfaces no pacote consumidor.

## Testes

- Teste unitário para cada regra de domínio; sem rede, Docker ou chaves reais em unitários.
- Fake provider; SQLite em memória (file::memory:?cache=shared); TestUnit_Behavior.

## TypeScript

- strict: true, sem any (unknown com guard); React PascalCase, hooks use.
- API exclusivamente pelo SDK gerado packages/sdk-ts; sem fetch manual para API Calango.
- Estilo por tokens packages/tokens; sem cores avulsas.

## Dart

- flutter analyze e dart format; SDK gerado packages/sdk-dart (CalangoClient + CalangoTransport); ThemeData de packages/theme-flutter (CalangoTheme.light/dark), composto só das constantes geradas de tokens.json. Sem chamadas HTTP manuais à API Calango nem cores avulsas. ADR 0024.

## API e banco

- Contrato primeiro em api/openapi.yaml, gerar SDKs, implementar.
- Rotas /v1/...; erros JSON com error.code e error.message; códigos estáveis: budget_exceeded, approval_required, not_found, invalid_input, unauthorized, node_offline.
- WebSocket /v1/events, eventos tipados (DATA_MODEL.md).
- Migrações numeradas em internal/store/migrations/NNNN_name.sql, imutáveis após commit; tabelas de domínio com id ULID, created_at, updated_at e owner_id quando aplicável.
