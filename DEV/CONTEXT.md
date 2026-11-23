# Current context

Updated: 2026-11-23.

- **Phase:** 0 (foundation). Core compiled and runnable with Go 1.27.4; the project's minimum version remains Go 1.22.
- **Exists and was verified:** `cmd/calango` (HTTP server + scheduler), `internal/config` (`CALANGO_ADDR` with loopback default and validation), `internal/core` (agent with budget and scheduler), `internal/api` (`/healthz`), core/config tests, CI with cross builds for six targets, Dockerfiles/compose, docs, OpenAPI draft, design tokens and site.
- **Does not exist yet:** database and migrations, gateway/provider, harness, desktop, mobile, Cloud, embedded transport, ADE and browser.
- **Decided:** Go, SQLite, Electron, Flutter, Tailscale as the initial transport, original code, `calango` binary, AGPL-3.0 with a commercial option (ADR 0010), English as the repository language.
- **Open decisions:** screen control in Pro or Team (D2); first harness (D3); domain/module name (D4); `tsnet` license/terms (D5); automatic Maestro update (D6). D3 does not block the SQLite store.
- **Research:** current competitive map in `DEV/COMPETITIVE_LANDSCAPE.md`; do not treat the PC+VPS+phone combination as an exclusive differentiator.
- **Next tasks:** phase 0 implemented locally; start F1-T01 (SQLite store) and F1-T02 (initial migrations). i18n generation/drift comes with F1-T26. See `DEV/HANDOFF.md`.
- **Recent local verification:** `go test`, `go vet`, format, six cross builds and `GET /healthz` passed. The GitHub run depends on a first push; Docker could not be checked in this session because of a daemon restriction.
