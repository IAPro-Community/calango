# Handoff

Updated: 2026-11-23.

## Confirmed state

- Go 1.27.4 installed in `~/.local/opt/go` (link at `~/.local/bin/go`).
- Phase 0: configuration and CI for six builds implemented. `go test`, `go test -race`, `go vet`, `gofmt`, six cross builds and a local health check passed (details in `DEV/VERIFY.md`).
- F0-T07: tokens generate CSS and TypeScript/Dart constants; `--check` runs in CI and passed locally. i18n generation/drift is left for F1-T26, when the catalog is created.
- The agent is still a scaffold: there is no SQLite store, gateway/provider, harness, desktop, mobile or cloud yet.
- Research and positioning: `DEV/COMPETITIVE_LANDSCAPE.md`. Technical delivery pipeline: `DEV/TECHNICAL_DELIVERY.md`. TODO updated: `DEV/TODO_V1.md`.
- Docker: build and health check history in `DEV/VERIFY.md`; not revalidated in this step because the sandbox denied access to the daemon. Bridge networking test is still pending.
- License AGPL-3.0 + commercial license (ADR 0010). CLA and contacts still need owner review before publication.
- Repository has no automatic commit; do not publish to `IAPro-Community/calango` without an explicit request.
- Language policy: the repository is in English (see `DEV/CONVENTIONS.md`). Public docs and `DEV/` are being translated to English; README is available in English, Spanish and Brazilian Portuguese.

## Next technical step

1. Start F1-T01/T02: `internal/store` with SQLite and initial immutable migrations. D3 (harness) does not block the store; it only decides the provider/harness items that depend on it.

## Owner decisions and publication limits

- D1b: legal review of `CLA.md`, `COMMERCIAL.md`, `TRADEMARKS.md`, legal name and contacts.
- D2 Pro/Team screen control; D3 first harness; D4 domain/module name; D5 `tsnet`; D6 automatic Maestro update.
- Run the workflow on GitHub after an authorized push; local lint and cross builds already passed. No commit/push without an explicit request.
- Full visual review of the site on phone and in dark mode; Docker bridge test on a compatible environment.
- Maestro: validate install destination, JSON format and non-interactive `update` per `DEV/ORQUESTRADOR_MAESTRO.md`.

## Language update (2026-11-23)

- Repository, public docs, `DEV/` and the OpenAPI descriptions are in English. README exists in English (`README.md`), Spanish (`README.es.md`) and Brazilian Portuguese (`README.pt-BR.md`). Contribution language rule is in `CONTRIBUTING.md#language`.
- The site has pt-BR, en and es. Strings are inline in `site/index.html` (dictionary `I18N`). The product catalog remains in `packages/i18n/messages.json`; the site does not read it yet.
- Open owner items: CLA legal name, contact emails (`licenca@`, `seguranca@`, `conduta@` at calango.dev), Caddy domain placeholder, Terms of Service link. Nothing was committed or pushed.
