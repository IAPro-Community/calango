# ADR 0022 — Skill import reads authorized roots and keeps versioned copies indexed in a file

Status: proposed · 2026-11-25 · issue #88 (F4-T07, foundation delivered ahead of phase 4)

## Context
Issue #88 asks Calango to discover Claude Code and Codex skills, show their origin, import them idempotently and respect permissions and licenses. A skill is a directory with a `SKILL.md` and supporting files under `~/.claude/skills`, `$CODEX_HOME/skills` or a project's `.claude/skills`. ADR 0014 already forbids Calango from writing into the person's `~/.codex` and `~/.claude`. Issue #124 will later propose skills from execution patterns into the `skills` table described in `DATA_MODEL.md`. At the time of this change another open PR owns migrations `0015` and `0016`, and no table for skills exists.

## Decision
1. Discovery reads only an explicit list of roots (`CALANGO_SKILL_DIRS`; defaults `~/.claude/skills` and `$CODEX_HOME/skills`). It never scans the disk, opens files read-only, follows symlinks only when they resolve inside their root, and stops at fixed size and count limits.
2. Imports are copies, not links: `<data-dir>/skills/store/<source_id>/<hash prefix>/`. The source can change or disappear without changing what Calango imported, and Calango never writes into the source.
3. The index of imported skills is a JSON file (`<data-dir>/skills/index.json`) written atomically, not a SQLite table. It holds metadata only; the content lives in the copies.
4. Identity is the source directory (`source_id`), versions are content hashes. The same hash is a no-op; a new hash replaces the copy only when the request says `replace`.
5. A skill without a declared license, or with a restrictive one, is imported only with explicit confirmation, which is stored on the copy and in the audit log.
6. Routes live under `/v1/skill-imports`, leaving `/v1/skills` for the approved skills of #124. Imported skills are not given to agent runs in this change.

## Consequences
- No migration: the feature does not collide with pending migration numbers, and the library can be dropped or rebuilt by deleting `<data-dir>/skills`.
- The index is not part of SQLite backups; `BACKUP_RESTORE.md` already copies the whole data directory offline, which includes it.
- Replication between nodes (ADR 0008) does not carry imported skills yet; each node imports from its own roots.
- When #124 adds the `skills` table, imported entries can be promoted with `source='import'`, keeping `source_id` and `hash` as provenance. Whether the file index then moves into SQLite is decided there.

## Alternatives rejected
- **Scan the home directory for `SKILL.md`.** Reads far more than the person authorized and has no bound.
- **Symlink to the source instead of copying.** The skill could change under a running agent and a link could point anywhere.
- **New SQLite table now.** Needs a migration number that another open PR is using, and #124 has not fixed the table's shape.
- **Write imported skills into `~/.claude/skills` or `~/.codex/skills`.** Contradicts ADR 0014.
