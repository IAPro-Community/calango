# ADR 0011 — Port Apache-2.0 code from AgenticOS into Calango's languages

Status: accepted · 2026-11-23 · amends [ADR 0006](0006-codigo-original.md) for one source only

## Context
The owner asked to reuse as much as possible of `vstorm-co/agenticos` (Apache-2.0, Python backend, Next.js frontend, Tauri desktop and mobile). ADR 0006 forbids any code, text, test, prompt or asset from reference projects. Apache-2.0 allows derivative works when the license, copyright notices and NOTICE attributions are kept and changes are marked.

Reference pinned at upstream commit `a4b0b3cd138bd6c64206c7039be859529775d87b` (`chore: cut 1.91.0`, 2026-11-23). Upstream is active, so every port records the file path and this commit.

Calango's stack does not change: Go core ([ADR 0001](0001-nucleo-em-go.md)), Electron + React + TypeScript desktop ([ADR 0003](0003-desktop-electron.md)), Flutter mobile ([ADR 0004](0004-mobile-flutter.md)). The Python runtime and the Next.js app are not ported as they are; their logic is rewritten.

## Decision
1. **Scope of the exception.** Only `vstorm-co/agenticos` at the pinned commit. Other references, such as agentes.chat, stay excluded by ADR 0006. Maestro stays under ADR 0009.
2. **What may be ported.** Logic, data model, API shape, validation rules, state machines and test scenarios, rewritten in Go, TypeScript or Dart. The port follows the behavior of the source, not its text.
3. **What is not ported.**
   - Prompts, system messages, skill and rule texts (`.claude/`, `AGENTS.md`, agent instructions): Calango writes its own text.
   - Brand marks and third-party assets bundled upstream (`brand-glyphs`, Chart.js, d3, Lucide artwork): each would need its own license check.
   - Tauri code, the sandbox runtime, channel bots, voice, workflows and virtual tables (V2+ or out of product scope).
   - Vstorm deployment, Docker, Traefik, Kubernetes and CI definitions.
   - Python dependencies of the upstream runtime (Pydantic AI and the rest). Calango's runtime stays in Go.
4. **Where a maintained library exists, use it instead of porting.** Example: MCP client via the official `github.com/modelcontextprotocol/go-sdk` (licence moving from MIT to Apache-2.0; checked before adding). Keychain access via OS libraries, after a licence check. Porting replaces only what no library covers.
5. **Attribution, required on every ported file.** Header line:
   `Derived from vstorm-co/agenticos (Apache-2.0) at a4b0b3c: <upstream path>. Modified.`
6. **Attribution, project-level.**
   - `NOTICE` at the root lists AgenticOS, copyright 2026 Vstorm, Apache-2.0, and each ported area.
   - The full Apache-2.0 text ships at `third_party/agenticos/LICENSE`, plus the upstream `NOTICE` text at `third_party/agenticos/NOTICE`.
   - Calango's own `LICENSE` (AGPL-3.0) stays as the license of the Calango code.
7. **Review rule (replaces the "reject copied fragments" rule of ADR 0006 for this source only).** A PR that ports code must list for each file the upstream path and commit, carry the header, and update `NOTICE`. A ported file without the header, or with an upstream path not listed in the PR, is rejected.
8. **Priority by roadmap (first to last).**

| Priority | Roadmap item | Upstream reference (lines, approx.) | Calango target |
|---|---|---|---|
| P1 | F1-T18 audit | `backend/app/core/audit.py` (283), `backend/app/services/audit.py` (351) | `internal/store` audit, `internal/audit` |
| P1 | F1-T10 vault | `backend/app/core/vault.py` (343) | `internal/vault` on OS keychains |
| P2 | F1-T08 approvals | `backend/app/services/approvals.py` (883), `backend/app/agents/approval.py` (143); UI `frontend/src/components/runs/approval-cards.tsx` (257), `workflow-approvals-card.tsx` (283), `chat/tool-approval-dialog.tsx` | `internal/approvals` extension; desktop approval card (F1-T23) |
| P2 | F1-T09 routines | `services/routine_actions.py` (505), `routine_delivery.py` (241), `routine_authority.py` (674), `routine_sources.py` (1106, partial) | `internal/routines` extension |
| P2 | F1-T11 memory | `repositories/memory.py` (291), `agents/memory_scope.py` (125), `schemas/memory.py` (121) | `internal/memory` extension |
| P2 | F1-T17 usage | `services/usage_report.py` (365); UI `chat/usage-meter.tsx` (488), `hooks/use-usage-stats.ts` (198) | `internal/usage`; desktop usage panel |
| P3 | F1-T12 MCP | `agents/mcp.py` (1040), `services/mcp_connection.py` (3655), `services/mcp_audit.py` (517) | `internal/tools` with go-sdk; scope logic and audit ported |
| P3 | F1-T23 screens | `dashboard/widgets/routines.tsx` (225), `components/runs/*`, chat components | Electron renderer (React + Vite) |

9. **Order of work.** One slice per PR, following the table. Each slice: port, header, `NOTICE` entry, tests that keep the upstream scenarios, and a `DEV/VERIFY.md` entry. Slices that depend on blocked items (D3 harness, conversation/run routes) wait for the owner's decision.

10. **Port checklist (per PR).**
    - Pin: the commit in each header is the one the logic was read at. Bump it only in a PR that re-reads the upstream diff for the files it touches.
    - Header on every new Go, TypeScript or Dart file derived from an upstream file; the PR description lists each upstream path.
    - `NOTICE` and `third_party/agenticos/` are updated by the first PR that lands an area.
    - Equivalence tests rewrite the upstream test scenarios in the target language; test code is not copied.
    - `DEV/VERIFY.md` records the commands run for the slice.

## Consequences
- Faster V1 delivery for P1–P3, at the cost of tracking upstream changes by hand (commit pinned per port).
- Apache-2.0 portions remain under Apache-2.0 inside the Calango repository. A third party can take those files without publishing, as ADR 0010 already records for Apache code. The core stays AGPL-3.0. Accepted, and it reinforces the legal review below.
- Ported code is not a clean-room rewrite. Anyone reviewing the port should check against the upstream file, not against memory.
- Open legal items (from ADR 0010): the legal review of `CLA.md` and `COMMERCIAL.md` must also cover Apache-derived files and the `NOTICE`.

## Discarded alternatives
- **Clean-room rewrite with no attribution:** does not meet Apache-2.0 for a derivative work; it would also be impractical when the goal is maximum reuse.
- **Run the upstream Python backend as a sidecar:** contradicts ADR 0001 (single Go binary) and adds Python to the PC node.
- **Copy the Next.js app as it is:** conflicts with ADR 0003 (Electron + Vite) and brings Next.js routing and server code.
