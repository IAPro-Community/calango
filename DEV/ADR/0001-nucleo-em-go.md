# ADR 0001 — Core in Go, single `calango` binary

Status: accepted · 2026-11-23

## Context
The same agent must run on the PC (Windows, macOS, Linux) and on the VPS (Linux, Docker), with low resource use and simple distribution.

## Decision
The whole core is Go, compiled into a single binary named `calango` (entry point in `cmd/calango`). PC and VPS run the same binary; only the configuration changes.

## Alternatives
- Node/TypeScript (as in the references): closer to the desktop, but worse in distribution and resource use on the VPS.
- Rust: excellent performance, steeper learning curve and slower iteration.

## Consequences
- Trivial cross-compilation (`GOOS`/`GOARCH`), minimal Docker image.
- Desktop (TS) and mobile (Dart) talk to the core only through the API; no duplicated logic.
- AI libraries in Go are less mature: the core talks to models via direct HTTP and via harness CLIs.
