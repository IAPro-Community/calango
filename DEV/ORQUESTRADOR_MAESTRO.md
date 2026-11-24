# Orquestrador Maestro inside Calango

Decision: [ADR 0009](ADR/0009-orquestrador-maestro.md). Module spec: `SPECS/modules/maestro.md`.

## What it is

A project of IAPro Community, Apache-2.0: <https://github.com/IAPro-Community/Orquestrador-Maestro>.
It organizes coding agents with rules, skill routing, compact context, verification and handoff.
Stable version of the V1 line: **1.1.1** (released 2026-10-18). npm package: `@iapro/orquestrador-maestro-cli`. Requires Node.js 20.19 or later.

Useful commands of the official CLI: `orquestrador-maestro install`, `verify`, `update`, `remove`, `sync`.

## How Calango uses it

1. **Installs** the V1 line in a folder of its own inside Calango, never in the user's personal home directory.
2. **Links** Maestro to the workspace of each agent the owner chooses (`AGENTS.md`, `CLAUDE.md`, or the harness equivalent).
3. **Verifies** each installation or update with `verify`. If it fails: rolls back.
4. **Follows** the npm registry and warns about new versions of the V1 line.
5. **Updates** according to the owner's preference (by card or automatically). Major version: only with a warning.

## Update flow

```
npm registry (1x/day)
   └─ new version in the V1 line?
        ├─ no → nothing
        └─ yes → download to a new folder (<calango-home>/maestro/versions/<v>)
              ├─ verify OK → switch the "current" pointer → keep the previous one
              │      ├─ automatic on → done, recorded in the Calango WORKLOG
              │      └─ off → card "Maestro 1.x.y available"
              └─ verify failed → keep the current pointer, warn
```

Rollback: `current` goes back to the previous version kept aside. Old versions are cleaned up after 3 successful installations.

## Rules

- The owner decides whether Maestro is on, per agent.
- Automatic update inside the V1 line is **on** by default (D6, ADR 0009 amendment). The owner can turn it off.
- Never update during a run in progress; wait for the run to finish.
- Changes to Maestro's rules do not alter the conversation history or the agent's memory.
- Calango does not edit Maestro's files by hand. It only installs, verifies and switches versions.
- Every update creates an entry in the audit log (`audit_log`) with the previous and the new version.

## Attribution

- `NOTICE` at the repository root and on the app's "About" screen.
- Link to the repository and to the Apache-2.0 license.
- Do not change the project's name or trademark.

## Technical open items (verify before implementing)

- Confirm whether `orquestrador-maestro install` accepts a target folder outside the default home. If not, use an isolated `HOME` per process.
- Confirm a machine-readable output (JSON) for `verify` and `update`.
- Confirm that `update` does not require interaction.
- Test on Linux, Windows and macOS: the official README cites all three.
