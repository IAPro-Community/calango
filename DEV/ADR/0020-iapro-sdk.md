# ADR 0020 — IAPro-SDK as a shared library

Status: proposed · issue #128 · first slice for human review.

## Decision

Consume the public module `github.com/IAPro-Community/IAPro-SDK` at the immutable revision `v0.0.0-20261124230439-ece304ad4225`, with checksums obtained by Go in SDK CI (run 45506771570). MIT: the license and NOTICE are preserved in the upstream repository; Calango's AGPL remains unchanged. Full provenance review and the legal gate remain pending. Do not copy modules between products.

The SDK provides HTTP status classification and the decision to advance within the agent's explicit chain. Calango's gateway remains the authority for budgets, audit, persistence and approval. Codex provider identity is canonical in the SDK. The SDK does not decide credentials, authorization, commercial policy, storage or execution. The configured chain is the existing fallback authorization; no provider outside it is selected. The CLI is not replayed after ambiguous failure. If tool or CLI side effects are possible, fallback proceeds only for pre-dispatch failures (not installed/signed out). Direct Anthropic does not execute tools; it preserves fallback before text is emitted.

Do not migrate authentication, quota probes, handoff, agent coordination, UI or subprocess runtime without evidence and dedicated integration. The SDK is not a mandatory daemon. Do not change CODEX_HOME, MCP, sandbox flags or OpenAPI contracts in this slice. Do not enable automatic selection based on missing quota evidence.

## Verification

Existing chain and harness tests, new TestSDKChainSafety, Anthropic tests and quality-gate. SDK and consumer CI must pass at the exact SHA; human review and integration follow GOVERNANCE.md. Real-provider acceptance and epic parity remain open. Rollback: revert the PR and associated pin. Do not declare v0.1 stable or close #128 before the remaining criteria are satisfied.

## Security review

The gateway allocates the usage identifier before dispatch; identifier, ledger or cost-calculation failures prevent fallback through ErrLedger. The adapter declares MayHaveSideEffects; absence of this capability is treated as possible side effects. The decision uses the resolved adapter's evidence rather than its name. Chain tests verify that backup is blocked without output when effects are possible and on ID failure. The complete MIT license and SDK NOTICE accompany the node image; future desktop installers must include them and verify the contents before publication (#39, #40, #116).

## Second slice — Codex runtime

Update the pin to `v0.0.0-20261125011527-b93285109b30`, with checksums and six successful jobs in SDK run 45524394245. `providers/codex.RunExec` and `CheckLoginStatus` replace local execution and decoding. The host retains existing sandbox/ephemeral flags, workspace, CODEX_HOME, MCP configuration, gateway, approvals and budget. `--json` emits complete messages (not token deltas). Terminal counters are used when both are measured; missing counts on success retain an explicit estimate; missing counts on failure return Usage{} so the gateway accounts for the reservation. No replay of ambiguous execution, token in argv, raw stderr or inherited prior bearer token. Sessions remain ephemeral. Probes, real quota, handoff, UI and persistent coordination are not enabled in this PR. Local fixtures do not prove real CLI compatibility (#53/#113). Rollback by reverting the PR/pin.
