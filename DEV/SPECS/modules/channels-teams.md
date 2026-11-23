# Module: channels-teams

Phase 6 · Packages `internal/conversations`, `internal/router`, `internal/members`

## Goal
Channels with people and agents; Team plan.

## Rules
- Channels with threads and mentions. A message without a mention goes to the **router**: a cheap model picks 0 to 2 agents. The router never grants permissions.
- The router's choice appears at the top of the reply it started, not as a loose line.
- "Nobody needs to reply" is a valid decision and appears discreetly.
- Members with roles (owner, admin, member, guest); permissions per channel.
- Delegated approval: an agent's owner can name a substitute.
- One agent can serve more than one person (shared specialist).

## Acceptance
- A message without a mention in a channel with three agents triggers at most two runs.
- A guest does not see channels they were not added to.
