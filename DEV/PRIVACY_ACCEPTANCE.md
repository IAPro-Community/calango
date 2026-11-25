# Privacy: scope and acceptance criteria (#46)

Status: proposed for PO review; does not represent a complete export or deletion implementation.

- Authenticated export of agents, non-secret settings, memories, conversations, routines, decisions, usage and metadata belonging to the data subject. Versioned JSON format, manifest and size limits. Exclude tokens, keys, encrypted envelopes, MCP credentials and other subjects' content.
- Deletion: inventory dependencies, cancel runs, prevent new work during the operation, revoke sessions and validate transactional deletion of permitted records. Confirm the scope of devices, accounts and connectors before removing them. Do not expose destructive deletion until audit retention and backups are defined.
- Append-only audit: decide which minimum metadata must be retained, the justification, retention period and pseudonymization procedure preserving verifiable integrity. Do not silently modify existing events.
- Backups and logs: define retention, expiration, restoration without reintroducing deleted data and operational responsibility. Local deletion does not imply deletion of data sent to external providers.
- First use: detect node, models, harness, vault and connectors locally; show only availability and authorized metadata, never secrets, prompts, results or raw environment. Do not send telemetry without consent.
- Acceptance tests: isolation between subjects and scopes, export without secrets, idempotent deletion, partial failure and recovery, token revocation, backup restoration, lack of caching and inspection of payloads/logs with synthetic fixtures.

Partial delivery in this PR: Cache-Control no-store on authenticated responses (including failures) and pairing. The header does not guarantee that the client cannot copy data. Export, complete deletion and initial discovery remain open and require scope approval and implementation.
