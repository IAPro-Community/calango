# Calango governance v1.0

Calango uses issue-driven GitHub Flow and incremental releases. This policy takes effect when the governance PR is accepted; repository settings must be independently verified.

## Roles and decisions

- Product Owner: vision, scope, backlog priority, release objectives and functional acceptance.
- Maintainers: repository policy, human review, merge and release authority.
- Technical leads: architecture and technical standards; significant decisions require a reviewed ADR.
- Contributors: implementation, testing and documentation of assigned issues.
- Reviewers: independent human review; authors must not approve their own work.
- AI agents: delegated execution under an accountable human; no independent approval, policy bypass or publication authority.

Anyone may propose changes. Scope, licensing, commercial policy and other owner decisions require the appropriate owners' approval. Record decisions in the issue and ADR where applicable. Maintainer identities and code ownership are established by the repository owners, not inferred from commits.

## Work and review

GitHub Issues owns work scope; GitHub Projects owns prioritization and workflow status; milestones own target deliveries. Specs and ADRs remain the technical source of truth. One primary issue per unit of work, one accountable human, and one active task per contributor/session by default. Use sub-issues for independently verifiable slices; dependencies must be explicit.

Use short-lived branches off main, linked PRs, a successful required CI gate, independent human approval, resolved conversations and squash merge. No direct pushes, force pushes, main deletion or permanent administrative bypass. Emergency policy changes require explicit owner authorization, an audit record and restoration. A green CI does not imply functional acceptance.

See [CONTRIBUTING](CONTRIBUTING.md), [engineering conventions](DEV/CONVENTIONS.md), [delivery process](DEV/TECHNICAL_DELIVERY.md) and [bootstrap settings](DEV/GOVERNANCE_BOOTSTRAP.md).

## Ready and done

Ready: objective, scope, acceptance criteria, dependencies, priority and no unresolved blocking decision. Done: acceptance evidence, relevant tests, required CI, independent review, merge and any required post-merge validation; update documentation when affected. Keep issues open until all criteria are met. A rejected or duplicate issue is closed with a reason, not counted as delivered.

## Releases

Use Semantic Versioning, v-prefixed immutable tags, optional alpha/beta/rc prereleases and a traceable main commit. Release scope is set by the Product Owner and maintainers; pre-1.0 releases do not change the approved V1 scope. Require changelog, compatibility, limitations, installation/update guidance, artifacts and verification appropriate to the milestone. Publishing requires maintainer authorization.

## Cadence

Continuous triage; review blockers and PRs when active; weekly refinement where useful; review milestone scope and release evidence. Measure team flow (lead/cycle time, throughput, WIP, blockers and review latency), not individual lines of code or commit counts.

Security reports follow [SECURITY.md](SECURITY.md). Contributions remain subject to [CLA.md](CLA.md) and the code of conduct.
