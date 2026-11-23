# ADR 0002 — SQLite por nó, sem cgo

Status: aceita · 2026-11-23

## Contexto
Cada nó guarda seus próprios dados e precisa funcionar offline. Instalação no PC não pode exigir servidor de banco.

## Decisão
Um arquivo SQLite por nó, via `modernc.org/sqlite` (Go puro, sem cgo). Modo WAL. Migrações versionadas e imutáveis em `internal/store/migrations/`. Acesso por interfaces no pacote `internal/store` para permitir Postgres na Calango Cloud gerenciada no futuro.

## Consequências
- Build sem toolchain C, cross-compile simples.
- Escrita concorrente limitada: um escritor por vez; adequado para uso pessoal.
- Replicação entre nós é por eventos (ADR 0008), não pelo banco.
