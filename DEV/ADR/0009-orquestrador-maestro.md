# ADR 0009 — Embedded Orquestrador Maestro, with automatic update through the official channel

Status: accepted · 2026-11-23 · **partially supersedes ADR 0006** (for Maestro only)

## Context
The Orquestrador Maestro (`IAPro-Community/Orquestrador-Maestro`, Apache-2.0) organizes the work of coding agents: rules, skill routing, compact context, verification and handoff. The owner wants Calango to embed this process and follow its updates. ADR 0006 forbids copying reference code; Maestro is the exception explicitly chosen by the owner, under a permissive license.

## Decision
1. **Do not copy the repository.** Calango consumes Maestro from the official npm package `@iapro/orquestrador-maestro-cli`, pinned to the **V1** line (`^1.1.1`). This way the code belongs to the official channel and updates arrive by version.
2. **Automatic update, with control.** `calango` checks the npm registry periodically (default: once a day). When a new version within the V1 line appears:
   - it applies it on its own only if the owner turned on "update automatically" (default: **off**, because an update changes the rules of every agent);
   - otherwise it shows a card: version, CHANGELOG summary and an **Update** / **Not now** button.
   - A major version (V2 or later) **never** goes in on its own: always a card with a breaking-change warning.
3. **Isolated installation.** Maestro is installed in a Calango folder (`<calango-home>/maestro/`), not at the user's root. It is linked only to the agents the owner chooses, per harness (Claude Code, Codex, etc.), through a link in the workspace's `AGENTS.md`/`CLAUDE.md`.
4. **Verification before activation.** After installing or updating, Calango runs `orquestrador-maestro verify`. If it fails, it reverts to the previous version and warns.
5. **Rollback.** Each installed version is kept; reverting means switching the version pointer.
6. **Attribution.** Apache-2.0 requires keeping the license and notices. Calango includes a `NOTICE` at the root and an "About" screen with credit to the project and a link to the repository.

## Discarded alternatives
- **Copy the code into the repo**: breaks automatic updates and becomes a fork to maintain.
- **Git submodule**: coupled to commits; manual update.
- **Install in the user's folder (`~/.orquestrador-maestro`, `~/.codex`)**: touches personal configuration outside Calango's control.

## Consequences
- Dependency on Node.js 20.19+ on the owner's machine (or inside the container, on the VPS). Enters in `F1-T03`/`F3-T01`.
- The Maestro version becomes part of the contract: when it changes, Calango shows the version in the agents panel.
- Anyone who does not want Maestro runs without it; nothing in the core depends on it.
