# Calango — active spec

Status: phase 0 (foundation). Code written from scratch. The Tucano and agentes.chat repositories are read-only references: no code, text or assets were copied.

## Persistent scope requested by the owner

Understand the product and the relevant files in the repository; map competitors using current sources; turn findings into ideas, implementation and technical documentation; keep the V1 TODO complete; verify each step and keep going until no required work remains. The competitive map and the technical pipeline are in `DEV/COMPETITIVE_LANDSCAPE.md` and `DEV/TECHNICAL_DELIVERY.md`. V1 is not finished yet.

## Goal

A personal assistant that runs on the computer and continues in the cloud: it executes tasks, routines (crons), channels with people and agents, its own browser, and an ADE for building separate projects. Clients: desktop, iOS, Android and web.

## Stack decisions (proposal, to be validated)

- Core and server: **Go**. A single binary, `calango`, runs locally and in the cloud.
- Database: **SQLite** via `modernc.org/sqlite` (pure Go, no cgo, easy cross-compilation). Postgres in the cloud comes later, behind an interface.
- API: HTTP + WebSocket (real time).
- Agent browser: **chromedp** (Chrome/Chromium via the DevTools Protocol).
- Desktop: **Electron** (Windows, macOS, Linux), which starts `calango` as a child process. Decided 2026-11-23.
- Mobile: **Flutter** (iOS and Android). Decided 2026-11-23. Shares with the desktop by contract: OpenAPI generates TypeScript and Dart SDKs; design tokens and translations have a single source.
- Full plan: [../PLANO.md](../PLANO.md).

## Phase 0 — scope of this delivery

- [x] Repo structure, `DEV/` docs.
- [x] Agent model with token budget.
- [x] Routine scheduler with an injectable clock (testable).
- [x] HTTP server with `/healthz`.
- [x] Build and tests verified with Go 1.27.4; `/healthz` returned 200 in a local run.
- [x] `internal/config`: `CALANGO_ADDR` centralized, validated before the scheduler starts; loopback default.
- [x] CI written for vet/test/format and linux/windows/darwin builds on amd64+arm64; local cross builds passed. The GitHub run depends on an authorized push.
- [x] OpenAPI semantic validator in CI; local validation passed with no problems or warnings.
- [x] Design token generation and drift checking in CI. The i18n catalog will be created in F1-T26.

## Transport between nodes (decided)

Tailscale in the initial phase, then an own relay. Details in [PRODUCT_HYBRID.md](PRODUCT_HYBRID.md).

## Out of scope for now

Channels, model integration, browser, ADE, mobile and desktop clients, cloud↔local synchronization.

## Acceptance criteria for phase 0

`go test ./...` and `go vet ./...` pass; `calango` starts and answers `GET /healthz` with 200; invalid configuration fails before starting; contract gates and supported builds pass in CI.
