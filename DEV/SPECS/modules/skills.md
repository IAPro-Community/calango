# Module: skills (import from Claude Code and Codex)

Phase 4 (F4-T07, issue #88), delivered early as a foundation · Package `internal/skills` · No SQLite table (ADR 0022)

## Goal

Find the skills the person already uses in Claude Code and Codex, show where each came from and under which license, and import a versioned copy into Calango without touching the originals.

## Skill format

A skill is a directory with a `SKILL.md` whose YAML frontmatter has `name`, `description` and optionally `license` and other fields. Supporting files (scripts, references) sit next to it and are part of the skill.

## Authorized roots

Only the directories in `CALANGO_SKILL_DIRS` are read (`DEV/SPECS/modules/config.md`). Unset, the defaults are:

| Origin | Path |
|---|---|
| `claude` | `~/.claude/skills` |
| `codex` | `$CODEX_HOME/skills`, default `~/.codex/skills` |

A project root (`<project>/.claude/skills`) is added explicitly as `project=<path>` until the ADE `projects` table exists; then project roots come from registered projects. Maestro's own Codex home (`<data-dir>/maestro/.codex`, ADR 0014) and Codex per-account homes (`<data-dir>/accounts/codex/<label>`) are Calango-managed and are not discovery roots.

## Discovery rules

- Read-only: files are opened for reading only; nothing under a root is written, renamed or re-moded.
- One level: each direct subdirectory of a root with a `SKILL.md` is one skill. Hidden entries (`.system` in Codex, which holds bundled skills) are not scanned. A subdirectory without `SKILL.md` is reported as skipped (`no_skill_md`).
- Symlinks: followed only when the resolved target is inside the resolved root. A skill directory linking out of the root is skipped (`symlink_outside_root`); a file inside a skill linking out is left out with the warning `symlink_skipped`. Directory symlinks inside a skill are never followed (no cycles).
- Limits (defaults): 256 skills per scan, 1024 entries per root, 128 files per skill, nesting depth 6, `SKILL.md` up to 256 KiB, 8 MiB per skill. A skill over a limit is skipped with a reason; a scan stopped by a limit sets `truncated`. A file that grows while it is read is an error, so a concurrent writer cannot push a skill past its limit.
- Per skill: `source_id` (stable hash of origin, root and entry name), name, description, origin, path, `hash` (`sha256:` over the sorted file list, sizes and contents), declared license and its source (`frontmatter`, `file` for `LICENSE`/`COPYING` next to `SKILL.md`, or `none`), warnings and `import_status` (`new`, `imported`, `update_available`).
- Frontmatter is parsed by a small subset reader (top-level scalars, quoted strings, block scalars, comments; nested values are skipped). Anchors, aliases, tags, tabs and duplicate keys make it `invalid_frontmatter`; the skill is still listed, named after its directory.

## Licenses

- Warnings: `no_license` (nothing declared) and `license_restrictive` (the declared text says proprietary, all rights reserved, confidential or no license).
- Either one sets `requires_license_confirmation`; the import is refused with `license_confirmation_required` unless the request carries `confirm_license: true`. The confirmation is recorded on the copy (`license_confirmed`) and in the audit log.
- License detection is a hint for the person, not legal advice; the license text travels with the copy when it is a file in the skill.

## Import

- `POST /v1/skill-imports {source_id, hash, confirm_license?, replace?}`. The client never sends a path: the daemon rescans the roots and finds the skill by `source_id`. A hash that no longer matches is refused with `skill_changed`.
- Copies go to `<data-dir>/skills/store/<source_id>/<first 24 hex of hash>/`, written in a temporary directory, checked against the hash from the bytes actually written, then renamed into place. Executable source files keep an owner-only exec bit (0700); other files are 0600.
- Idempotent: the same hash returns `result: unchanged` with no write. A different hash needs `replace: true` (otherwise `replace_required`); the new version is activated by the index write, then the old version directory is removed.
- Index: `<data-dir>/skills/index.json`, written atomically (temporary file, fsync, rename). Ids and versions are checked against their hex shape on every read, so a tampered index cannot point outside the library. At startup, temporary files and copies the index does not reference are swept.
- Remove: `DELETE /v1/skill-imports/{id}` removes the index entry and the copy; the source is untouched.
- Audit: `skill.import`, `skill.replace`, `skill.remove` with name, origin, hash, license and confirmation (no file contents).

## API and permissions

| Route | Scope |
|---|---|
| `GET /v1/skill-imports/discover` | read |
| `GET /v1/skill-imports` | read |
| `POST /v1/skill-imports` | write |
| `DELETE /v1/skill-imports/{id}` | write |

The namespace `/v1/skill-imports` is separate from `/v1/skills` in `memory.md`, which stays reserved for the owner-approved skills of issue #124.

## Hook for agents (not wired)

Imported skills are not given to agent runs yet. `Library.Dir(ctx, id)` returns the directory of the active copy (with its `SKILL.md`) and is the intended entry point. Open, before wiring:

- How a run receives skills: inject the `SKILL.md` text into the prompt under a token budget, or point the CLI harness at a skills directory (for Codex, a per-run `CODEX_HOME/skills` assembled from the library; for Claude Code, `.claude/skills` in the run workspace).
- Which agents get which skills (per-agent grant, like connector grants).
- Issue #124 (skills proposed from execution patterns): proposals land in the `skills` table with `source='agent'`; an imported skill can be promoted into that table with `source='import'`, keeping `source_id` and `hash` as provenance. Imported content is third-party text: it must enter runs marked as data and pass the same secret and malicious-instruction checks #124 requires.

## Desktop

Screen "Skills": roots and their state, discovered skills with origin, license, warnings and import/update buttons (with the license confirmation checkbox when needed), skipped entries with reasons, and imported skills with removal. Texts in en, pt-BR and es.

## Tests

`internal/skills/skills_test.go` (frontmatter subset; discovery with origins, licenses and warnings; symlinks out of the root; limits; cancellation; idempotent import; update with replace; license confirmation; removal; source tree unchanged byte for byte, mode and mtime; sweep of leftovers; malformed index), `internal/api/skills_test.go` (lifecycle and error codes; token and scope checks), `internal/config/skills_test.go` (defaults, `CODEX_HOME`, explicit list, malformed entries).
