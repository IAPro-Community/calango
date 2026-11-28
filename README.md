<p align="center">
  <img src="site/assets/logo-192.png" width="112" alt="Calango logo" />
</p>

<h1 align="center">Calango</h1>

<p align="center">
  <strong>Your personal assistant. It runs on your computer, keeps working on your VPS, and answers from your phone.</strong>
</p>

<p align="center">
  Routines · approvals · browser · ADE for building projects · channels with people and agents
</p>

<p align="center">
  <a href="LICENSE"><img alt="License: AGPL-3.0" src="https://img.shields.io/badge/license-AGPL--3.0-E8A600" /></a>
  <img alt="Status: under construction" src="https://img.shields.io/badge/status-under%20construction-2D7A22" />
  <img alt="Go 1.25+" src="https://img.shields.io/badge/go-1.25%2B-00ADD8" />
</p>

<p align="center">
  <b>English</b> · <a href="README.es.md">Español</a> · <a href="README.pt-BR.md">Português (BR)</a>
</p>

---

> [!NOTE]
> **Under construction.** We are building Calango V1 in public. See [GitHub Issues](https://github.com/IAPro-Community/calango/issues), the [V1 planning snapshot](DEV/TODO_V1.md), and [how to contribute](CONTRIBUTING.md).

## Why Calango

- **It remembers, runs and keeps what you ask for.** It works on the computer you already use.
- **Nothing important happens without your OK.** Sending, deleting, publishing, paying or deploying always waits for an approval.
- **Your data stays with you.** Each node keeps its own database. Secrets never reach clients.
- **Control it from anywhere.** Use the desktop app, the phone, or both. Your VPS keeps working while your computer sleeps.

## Quick start

### See the project page (Docker)

```bash
docker build -t calango-site .
docker run --rm -p 8080:8080 calango-site
```

Then open <http://localhost:8080>.

### Run the core (Go 1.25+)

```bash
go vet ./...
go test ./...
go run ./cmd/calango          # listens on 127.0.0.1:7317
curl http://127.0.0.1:7317/healthz
```

### Build the node image (for a VPS)

```bash
docker build -f deploy/Dockerfile.node -t calango/calango .
```

An example compose file with automatic TLS is in [deploy/docker-compose.yml](deploy/docker-compose.yml).

## Repository layout

| Path | Contents |
|---|---|
| `cmd/calango` | server entry point |
| `internal/` | core: agents, scheduler, API |
| `api/` | OpenAPI contract (source of the SDKs) |
| `packages/tokens` | design tokens (source of CSS and the Flutter theme) |
| `packages/i18n` | three-language catalog (TypeScript and ARB for Flutter) |
| `site/` | project page |
| `deploy/` | node Dockerfile, compose file, Caddy |
| `DEV/` | product and engineering documentation |

## Generated files

Files under `packages/tokens/generated` and `packages/i18n/generated` are derived from the JSON sources. Regenerate them after editing a source and check for drift:

```bash
node scripts/generate-tokens.mjs
node scripts/generate-i18n.mjs
node scripts/generate-tokens.mjs --check
node scripts/generate-i18n.mjs --check
```

## Documentation

- [docs/guia](docs/guia/README.md): user guide in Brazilian Portuguese (getting started, pairing, VPS, Maestro and privacy).

- [GOVERNANCE.md](GOVERNANCE.md): branch policy and decision authority.

- [AGENTS.md](AGENTS.md): entry point for anyone writing code here, human or AI agent.
- [DEV/INDEX.md](DEV/INDEX.md): map of all documentation.
- [DEV/PLANO.md](DEV/PLANO.md): full product overview.

## Language policy

Brazilian Portuguese is the team's operational language. Technical identifiers and contracts remain stable. See [CONTRIBUTING.md](CONTRIBUTING.md#idioma).

## License

Open source under the [GNU AGPL-3.0](LICENSE). A commercial license is available for those who cannot comply with the AGPL: see [COMMERCIAL.md](COMMERCIAL.md). Contributions follow the [CLA](CLA.md). Name and logo rules: [TRADEMARKS.md](TRADEMARKS.md).

Calango uses the [Orquestrador Maestro](https://github.com/IAPro-Community/Orquestrador-Maestro) (Apache-2.0) as the work process for its agents. Credits and notices are in [NOTICE](NOTICE).

## Security

See [SECURITY.md](SECURITY.md). Do not open a public issue for a vulnerability.
