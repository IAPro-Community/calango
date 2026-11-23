# Code conventions

## General

- **Repository language is English.** Code, identifiers, comments, commit messages, issues, pull requests, `DEV/` docs, ADRs, specs and the public docs are written in English.
- The product interface ships in three languages: English (default), Brazilian Portuguese (`pt-BR`) and Spanish (`es`). Every user-facing string goes through the catalog in `packages/i18n`. No hard-coded text on screen.
- Locale detection order: explicit user choice > saved preference > browser/OS language (first supported match) > `en`.
- Small imperative commits; PR title and squash commit use Conventional Commits: `type(scope): description`, with `!` and BREAKING CHANGE for incompatible changes. Scopes follow modules (core, api, store, config, agents, gateway, harness, memory, routines, approvals, mcp, maestro, auth, vault, audit, pairing, desktop, mobile, web, cloud, ade, browser, terminal, teams, billing, licensing, ci, deploy, release, deps).
- Issue-linked short-lived branches; mandatory independent human review and CI before squash merge. See [governance](../GOVERNANCE.md).
- Never commit secrets (`.env` stays out of git).

## Go (`cmd/`, `internal/`)

- Go 1.22+. `gofmt` and `go vet` clean. `staticcheck` once CI exists.
- `internal/core` is pure domain: no network, no disk, no global clock (injected clock).
- Errors with context (`fmt.Errorf("routine %s: %w", id, err)`); exported sentinel errors for what the caller must distinguish.
- `context.Context` as the first parameter in everything that does I/O.
- No panics in production code, except for impossible invariants.
- External dependencies only with a written reason in the PR. Prefer the standard library.
- Interfaces are declared in the package that consumes them, not in the one that implements them.

## Tests

- Unit test required for every domain rule.
- Tests do not depend on the network, Docker or real keys. The AI model is a fake in tests (fake provider).
- Database in tests: in-memory SQLite (`file::memory:?cache=shared`).
- Name: `TestUnit_Behavior`.

## TypeScript (desktop, web)

- `strict: true`, no `any` (use `unknown` with a guard).
- React with PascalCase components, hooks prefixed with `use`.
- API client: only the generated one (`packages/sdk-ts`). Never a hand-written `fetch` to the `calango` API.
- Style: tokens from `packages/tokens`; no loose colors in CSS.

## Dart (mobile)

- `flutter analyze` clean, `dart format`.
- API client: only the generated one (`packages/sdk-dart`).
- Theme: `ThemeData` composed from the Dart constants generated from `packages/tokens`.

## API

- Contract first: change `api/openapi.yaml`, regenerate the SDKs, then implement.
- Versioned routes: `/v1/...`. Errors as JSON `{ "error": { "code": "...", "message": "..." } }`.
- Error codes are words from a fixed list (`budget_exceeded`, `approval_required`, `not_found`, `invalid_input`, `unauthorized`, `node_offline`).
- Real time over WebSocket at `/v1/events` with typed events (see `DATA_MODEL.md`).

## Database

- Numbered migrations in `internal/store/migrations/NNNN_name.sql`, immutable once committed.
- Every domain table has `id` (ULID text), `created_at`, `updated_at` and `owner_id` where applicable.
