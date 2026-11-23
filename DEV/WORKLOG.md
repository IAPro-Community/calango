# Worklog

## 2026-11-22 — foundation (phase 0)

- Reference scan: Dot (closed), Grok Bot, Tucano, agentes.chat. Read README, CHANGELOG, ADRs, open issues and PRs. Clones stay outside the project, as reference only.
- Decided: core in Go, one `calango` binary; pure-Go SQLite; Wails for desktop; native mobile consuming the API (Flutter vs React Native still open at the time).
- Created skeleton: `go.mod`, `internal/core` (agent with budget, routine scheduler), `internal/api` (`/healthz`), `cmd/calangod`.
- Tests written, not run: Go was not installed on this machine.
- Decided: node-to-node transport via Tailscale in the initial phase (evaluate tsnet for embedding). Own relay later. Recorded in `DEV/PRODUCT_HYBRID.md`.
- 2026-11-23: full plan written in `DEV/PLANO.md` (vision, topology, stack, 17 modules, security, data, monetization, distribution, monorepo, 8-phase roadmap, lessons from references).
- 2026-11-23: desktop becomes Electron (previously Wails). Binary renamed from `calangod` to `calango` (`cmd/calango`), in code, Docker and docs.
- 2026-11-23: mobile decided as Flutter. Plan adjusted: OpenAPI generates TypeScript and Dart SDKs; tokens and translations with a single source; monorepo gains `api/`, `sdk-dart/` and `tokens/`.
- 2026-11-23: complete technical documentation so another agent can build it: `AGENTS.md` at the root, `DEV/INDEX.md`, `CONTEXT.md`, `CONVENTIONS.md`, `SECURITY.md`, `DATA_MODEL.md`, 8 ADRs, `SPECS/ROADMAP.md` with numbered tasks (F0–F7), 20 module specs in `SPECS/modules/`, `api/openapi.yaml` (draft v0.1), `packages/tokens/tokens.json` (palette sampled from the logo), `DEV/BRAND/BRAND.md`.
- 2026-11-23: presentation site in `site/index.html` (static): hero with a sticky-note wall and glass tile, product demo driven by a single clock (5 scenes, pause/drag/skip), animated PC↔VPS↔phone diagram, features, ADE with code and typing terminal, security, plans, platforms, roadmap, FAQ. Light/dark and `prefers-reduced-motion`. Logo copied to `DEV/BRAND/logo-original.png` and resized into `site/assets/`.
- 2026-11-23: Go 1.27.4 installed without sudo in `~/.local/opt/go` (`~/.local/bin/go`), SHA-256 checked against go.dev. `go vet`, `go test` (4 tests) and `gofmt` pass. `calango` starts and `/healthz` returns 200.
- 2026-11-23: Orquestrador Maestro (Apache-2.0) incorporated by ADR 0009: consumed from the npm package `@iapro/orquestrador-maestro-cli` on the V1 line (1.1.1), update by card or automatic (default off), `verify` and rollback. Documented in `DEV/ORQUESTRADOR_MAESTRO.md`, `SPECS/modules/maestro.md`, `NOTICE`. ADR 0006 gained an exception only for Maestro.
- 2026-11-23: `DEV/TODO_V1.md` created with all V1 tasks (roadmap phases 0–6 + Maestro + cross-cutting) and decisions D1–D6 that block parts of the delivery.
- 2026-11-23: owner decided that Calango code will be open source (D1). The specific license was still to be chosen (D1a); recommendation: Apache-2.0. No `LICENSE` had been created yet.
- 2026-11-23: license decided: AGPL-3.0 + commercial license + CLA + trademark policy (ADR 0010). Created `LICENSE` (official text, gnu.org), `COMMERCIAL.md`, `CLA.md` (draft for review), `TRADEMARKS.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `.gitignore`, `.dockerignore`, `.github/workflows/ci.yml`.
- 2026-11-22: review of the site (`site/index.html`). Contrast measured in the browser: green titles and eyebrows, primary button, roadmap numbers, ADE cards and tags in dark mode were below AA. Fixed with text tokens (`--moss-text`, `--btn`, `--num`, `--warn-text`, `--on-tone`). Serious bug fixed: the ADE section title was invisible in dark theme. Copy trimmed (leads, features from 12 to 8, security from 6 to 4 cards). New effects: capabilities tape, progress bar, marker drawn on titles, stage tilt with the mouse, tilt and hover on cards, all under `prefers-reduced-motion`.
- 2026-11-22: attempt to validate Docker blocked by the `docker` group. After the owner added the user to the group, both builds passed (site and node). Health checks of the site and node responded 200 with `--network host`. Bridge networking does not work on this machine (`operation not supported`).
- 2026-11-23: root Dockerfile became the **site** (nginx non-root, port 8080, healthcheck, CSP). Node Dockerfile moved to `deploy/Dockerfile.node`. "Contribute to V1" section added to the site.
- 2026-11-23: local git initialized on `main`, **no commit**. Nothing published.
- Deployment model: server on the person's VPS via Docker, clients connect over TLS. Documented in `DEV/DEPLOYMENT.md`. Dockerfile, compose and Caddyfile created, not tested.

## 2026-11-23 — harden the foundation and consolidate V1

- `internal/config` centralizes `CALANGO_ADDR`, keeps loopback by default and validates host/port before the scheduler; invalid configuration test added.
- `core.Agent.Charge` avoids `int64` overflow; regression test guarantees that a refused charge does not change usage.
- CI includes cross builds for Linux/Windows/macOS on amd64/arm64; the six builds, tests, race, vet and format passed locally. `GET /healthz` returned 200. The workflow still depends on a run on GitHub after an authorized push.
- Research with official sources in `DEV/COMPETITIVE_LANDSCAPE.md`; plan repositioning and approval, idempotency, unknown cost and evidence criteria adjusted in the specs and the V1 TODO.
- OpenAPI 3.1 lint added to CI with Redocly CLI MIT 2.81.0 pinned; operationIds and tag descriptions completed. Local lint passed with no warnings.
- `DEV/TECHNICAL_DELIVERY.md` created for the CI/release/migration/rollback flow. Docker was not revalidated: the sandbox did not allow access to the daemon.
- Deterministic token generator creates CSS and TypeScript/Dart constants; CI checks drift. `node scripts/generate-tokens.mjs`, `--check` and `node --check` passed. i18n was split out to F1-T26, together with the catalog creation.
- Next: F1-T01 (SQLite store) and F1-T02 (initial migrations); they do not depend on D3, which blocks only the provider/harness.
- 2026-11-23: repository language policy set to English. Contribution rules updated in `CONTRIBUTING.md`, `AGENTS.md` and `DEV/CONVENTIONS.md`. Public docs (`README` in English, Spanish and pt-BR, `CLA`, `COMMERCIAL`, `TRADEMARKS`, `SECURITY`, `CODE_OF_CONDUCT`, `NOTICE`) translated. Translation of `DEV/` in progress.
- 2026-11-23: translation of `DEV/` finished (ADRs, module specs, specs, architecture, security, deployment, data model, monetization, brand, competitive landscape). Filenames with Portuguese slugs kept so links keep working.
- 2026-11-23: site `site/index.html` now ships in English (default), Brazilian Portuguese and Spanish. Detection order: `?lang=` > `localStorage` `calango-lang` > `navigator.languages` (pt/en/es) > `en`. Language switcher in the nav; `<html lang>`, title, meta description and hreflang alternates follow the locale. Also fixed the rotating hero word clipping descenders.
