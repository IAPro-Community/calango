# Contributing to Calango

Thank you for wanting to help. Calango is a personal assistant that runs on your computer and on your VPS. We are building **V1** in public.

Before anything else, read:
- [LICENSE](LICENSE) (AGPL-3.0)
- [CLA.md](CLA.md): you must accept the contributor agreement in your first pull request
- [TRADEMARKS.md](TRADEMARKS.md): rules for using the name and logo
- [AGENTS.md](AGENTS.md): rules for contributing code, including with AI agents
- [DEV/TODO_V1.md](DEV/TODO_V1.md): what is left for V1

## Language

**The project is in English.** To keep it readable for everyone, please write in English:

- issues and their titles and descriptions;
- pull requests, their titles and descriptions, and review comments;
- commit messages;
- code, identifiers, code comments and documentation (including `DEV/`);
- user-facing strings go through the translation catalog (`packages/i18n`), which holds English, Brazilian Portuguese and Spanish.

Pull requests and issues written in another language may be asked to be translated before review. Reading and understanding are not a barrier: if English is hard for you, write in your language and a maintainer or a translation tool can help, but the final text should be in English.

Translated versions of the README are available in [Español](README.es.md) and [Português (BR)](README.pt-BR.md).

## Where to start

1. Pick an open task from the [V1 TODO](DEV/TODO_V1.md) or the [roadmap](DEV/SPECS/ROADMAP.md).
2. If the task has an identifier (e.g., `F1-T03`), use it in the title of your PR.
3. For a large change, open an issue first with the proposal. Architecture decisions become ADRs in `DEV/ADR/`.
4. Not sure where to start? Items marked "good first issue" are small and well defined.

## Development environment

Requirements: Go 1.23 or later. Node.js 20.19 or later for the Orquestrador Maestro.

```bash
go vet ./...
go test ./...
go run ./cmd/calango          # listens on 127.0.0.1:7317
```

Presentation site:

```bash
docker build -t calango-site .
docker run --rm -p 8080:8080 calango-site
# open http://localhost:8080
```

## Code rules

- Follow [DEV/CONVENTIONS.md](DEV/CONVENTIONS.md).
- API changes start in `api/openapi.yaml`.
- Database changes are a new migration. Committed migrations are never edited.
- A sensitive action must never run without the owner's approval (see [DEV/SECURITY.md](DEV/SECURITY.md)).
- Do not submit secrets, keys, personal data or logs containing conversation content.
- Do not copy code from other projects. See [ADR 0006](DEV/ADR/0006-codigo-original.md). The Orquestrador Maestro exception is in [ADR 0009](DEV/ADR/0009-orquestrador-maestro.md).

## Pull requests

- One subject per PR.
- Tests for domain rules.
- Describe what changed, why, and how to verify it.
- Update the documentation in `DEV/` when the change affects scope, architecture or the API.
- Accept the CLA when the bot asks.

## Code of conduct

We expect respect and collaboration. Anyone who does not follow this may be removed from the project. Report problems as described in [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Security

Do not open a public issue for a vulnerability. See [SECURITY.md](SECURITY.md).
