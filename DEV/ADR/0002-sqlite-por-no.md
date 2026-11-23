# ADR 0002 — SQLite per node, no cgo

Status: accepted · 2026-11-23

## Context
Each node stores its own data and must work offline. Installing on the PC must not require a database server.

## Decision
One SQLite file per node, via `modernc.org/sqlite` (pure Go, no cgo). WAL mode. Versioned and immutable migrations in `internal/store/migrations/`. Access through interfaces in the `internal/store` package, to allow Postgres in the managed Calango Cloud in the future.

## Consequences
- Build without a C toolchain; simple cross-compilation.
- Limited concurrent writes: one writer at a time; adequate for personal use.
- Replication between nodes happens through events (ADR 0008), not through the database.
