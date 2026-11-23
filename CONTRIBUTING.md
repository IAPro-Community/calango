# Contributing to Calango

Thank you for wanting to help. Calango is a personal assistant that runs on your computer and on your VPS. We are building **V1** in public.

Before anything else, read:
- [LICENSE](LICENSE) (AGPL-3.0)
- [CLA.md](CLA.md): you must accept the contributor agreement in your first pull request
- [TRADEMARKS.md](TRADEMARKS.md): rules for using the name and logo
- [AGENTS.md](AGENTS.md): rules for contributing code, including with AI agents
- [GOVERNANCE.md](GOVERNANCE.md): roles, decisions and workflow
- [GitHub Issues](https://github.com/IAPro-Community/calango/issues): operational work; legacy V1 TODO remains historical until migration is verified

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

1. Find or propose an issue in [GitHub Issues](https://github.com/IAPro-Community/calango/issues). Check existing work to avoid duplicates.
2. Ask for assignment and confirm Ready status before implementation. Keep the legacy task ID in the issue when available.
3. Update main locally; create a short-lived branch `<type>/<issue-number>-<short-description>` from main. Types: feat, fix, refactor, docs, test, ci, chore, spike, hotfix. Use forks for external contributions and isolated worktrees for parallel sessions.
4. Open a draft PR linked to the issue early; test, request independent human review, resolve conversations and pass CI.
5. Maintainers squash merge after all gates. Use Refs #N for partial deliveries; Closes #N only if acceptance is complete at merge. Delete the branch after merge.
6. Significant architecture decisions require a reviewed ADR in DEV/ADR/. Use good first issue for suitable starter tasks.

Never push directly to main. See [GOVERNANCE.md](GOVERNANCE.md) for Ready/Done and decision authority.

## Development environment

Requirements: Go 1.22 or later. Node.js 20.19 or later for the Orquestrador Maestro.

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
- Accept the CLA in your first PR; follow the documented maintainer process until automation is verified.
- PR titles use Conventional Commits (e.g. `feat(vault): rotate master keys`). The squash commit must retain this format.
- Do not approve your own work. Disclose material AI assistance; report actual validation and any unrun checks.

## Code of conduct

We expect respect and collaboration. Anyone who does not follow this may be removed from the project. Report problems as described in [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Security

Do not open a public issue for a vulnerability. See [SECURITY.md](SECURITY.md).
