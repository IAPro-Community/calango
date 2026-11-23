# Governance bootstrap — issue #1

This is the implementation and administrative verification checklist. It is not evidence that repository settings have already been changed.

## Repository administrator checklist

- [ ] Confirm actual maintainers, independent human reviewer(s), and appropriate write permissions. Add .github/CODEOWNERS using verified GitHub usernames/teams only; enable code-owner reviews where appropriate.
- [ ] Settings > General: enable squash merge only; use PR title for squash commit; delete head branches automatically; enable auto-merge if desired (only with all checks and approval); verify Issues and Discussions.
- [ ] Settings > Rules > Rulesets: active ruleset protected-main targeting main; prevent deletion and non-fast-forward updates; require PR, one independent approval, dismiss stale approvals, require resolution of conversations, linear history, squash only, no permanent bypass.
- [ ] Verify the CI workflow actually runs on a PR and identify the exact quality-gate check before requiring it. Require that check and up-to-date branches. Do not merge bootstrap until the human approval and gate are verified.
- [ ] Confirm private security reporting/contact and CLA acceptance process before relying on either as an automated gate. Do not claim that a CLA bot exists until verified.
- [ ] Configure repository labels, organization GitHub Project and milestone strategy below.
- [ ] Record links/screenshots or API results for settings, CI run, human review and eventual squash merge in issue #1. No self-approval. Keep #1 open until post-merge verification.

## Project configuration

Create a single organization project called Calango Product. Add repository issues (including #1) and enable auto-add for new Calango issues, if available. Use one status field: Backlog, Refinement, Ready, In Progress, In Review, Validation, Done. Mark blocked via dependency plus blocked label; preserve actual status. Priority field P0 critical, P1 next milestone blocker, P2 important, P3 later. Size XS (<2h), S (half day), M (2 days), L (5 days), XL (split before implementation); indicative human effort, not commitments. Use native assignees, milestone and labels for area/type instead of duplicate fields. Views: Development Board, Product Roadmap, Bugs, Owner Decisions, Release Planning, My Tasks. Configure automation for new issues and PRs without treating every merge as Done; close only after acceptance.

Labels (create with descriptions): type:epic, type:feature, type:bug, type:task, type:decision, type:spike, type:docs; area:core, area:desktop, area:mobile, area:cloud, area:ade, area:browser, area:security, area:teams, area:devops, area:website, area:product; good first issue, help wanted, needs triage, blocked, breaking change. Do not duplicate priority, size or status in labels.

Proposed milestones, not existing releases: v0.1.0 Core Foundation; v0.2.0 Local Assistant; v0.3.0 Connected Assistant; v0.4.0 Hybrid Assistant; v0.5.0 Developer Workspace; v0.6.0 Browser & Automation; v0.7.0 Collaboration; v1.0.0 General Availability. Confirm scope and actual tags before creation. Do not import functional TODO items in this bootstrap.

## CI evolution

Extend existing workflow, do not replace product checks. Add stable aggregate quality-gate that fails if any required dependency does not succeed (including skipped/cancelled). Future gates are introduced when appropriate: Staticcheck, race tests, govulncheck, CodeQL, secret/dependency scanning, desktop/mobile tests, signing and release smoke tests. Use minimum permissions and no secrets in untrusted fork PRs.

## Backlog transition

DEV/TODO_V1.md and roadmap checkboxes remain intact pending reconciliation. For now do not silently drop legacy tasks or duplicate them into issues; use issue #1 for governance only. After reviewed migration, retain original IDs and evidence and make GitHub the operational backlog. Do not require every parallel contributor to update global ACTIVE/HANDOFF/VERIFY/WORKLOG. Use issue/PR evidence; update shared specifications only when actually changed.

## Evidence

Baseline commit dee8efaab2434c1145b4b96118af2edc9a1922f1; governance issue #1; branch chore/1-governance-bootstrap. CI and administrator settings must be verified separately. Final evidence belongs in the issue and PR, not in an assertion in this file.
