# Current context

Updated: 2026-11-25.

- **Phase:** 1 (local desktop MVP) in progress; Phase 0 complete. Phase 1 exit is not met; do not start Phases 2–6 until it is.
- **Core verified:** Go daemon; immutable SQLite migrations `0001`–`0014`; vault/auth; gateway and Anthropic/Codex providers; agents; conversations/runs; approvals; routines; memory plus `memory_save` and purge scheduler; MCP gateway; audit verify/export; usage; pairing/devices; `/v1/events`.
- **Desktop beta:** sandboxed Electron with agents, conversations, approvals, routines, usage, settings and audit. Fake-provider flows and mocked approval card pass. Linux Electron smoke verified settings, audit-chain verification/export and XDG startup entry create/remove. No real model call or external action.
- **Open Phase 1 gates:** Windows installer/signing, authorized real Codex/MCP approval action, real provider pricing, OS keyring runtime on Windows/macOS, Maestro stable-home gap, and acceptance across supported platforms. Reconnect state recovery is through REST; WebSocket replay is intentionally omitted by ADR 0015.
- **Decided:** Go, SQLite, Electron, Flutter, Tailscale as initial transport (D5 legal clearance still required before use), `calango` binary, AGPL-3.0 plus commercial option, English repository language.
- **Reuse:** AgenticOS (Apache-2.0) ports follow ADR 0011 with per-file attribution; audit chain and vault envelope were ported.
- **Open owner decisions:** D1b legal review/name/contacts; D5 written Tailscale clearance or alternate transport; pricing/domain/Terms and website claims; permission to delete `/tmp/agenticos`. D3, D2, D4 and D6 are answered.
- **Research:** competitive map is in `DEV/COMPETITIVE_LANDSCAPE.md`; D5 findings and sources are in ADR 0005.
- **Next:** resolve the Maestro `CODEX_HOME` mismatch with ADR 0014; finish remaining Phase 1 acceptance only with owner authorization/platform access; do not begin Phase 2–6 before the Phase 1 exit.
- **Recent verification:** `go test -race ./... -count=1 -timeout 300s`, `go vet ./...`, desktop typecheck/build, i18n drift check and `npm run test:main -w apps/desktop` passed. Isolated Linux Electron smoke used `/tmp` profile/data/key and no model call; site reviewed at desktop/mobile widths in light/dark. No commit or push.
- 2026-11-25: Docker site/node builds and a named-bridge smoke passed. The node database remained available after container recreation with a named volume. Compose/Caddy/TLS was not exercised.
