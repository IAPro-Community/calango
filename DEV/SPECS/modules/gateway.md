# Module: gateway

Phase 1 · Package `internal/gateway` · Table `usage` · ADR 0007

## Goal
Single entry point for every model call: budget before, usage after.

## Interface (Go)
```go
type Provider interface {
    Name() string
    Complete(ctx context.Context, req Request, stream func(Chunk)) (Usage, error)
}
type Gateway struct { /* store, providers, prices, clock */ }
func (g *Gateway) Call(ctx context.Context, who Caller, req Request, stream func(Chunk)) (Usage, error)
```
`Caller` = agent, routine (optional), project (optional), node.

## Rules
1. Before: the agent's month total + the estimate for the call ≤ limit. Otherwise `budget_exceeded` and nothing goes out.
2. After: records `usage` with tokens, price source and cost state (`known`, `estimated` or `unknown`). `cost_micros` is a nullable integer; do not use floats.
3. Emits `usage.threshold` at 80% and 100%.
4. The price table per model is versioned in code; unknown pricing never becomes zero cost. Zero cost is valid only when confirmed by the provider/plan. If there is a hard monetary ceiling and the cost cannot be estimated, block the call.
5. No package outside the gateway imports a provider SDK.
6. The panel distinguishes known value, estimate and absence of price; it never presents an estimate as a billed amount.

## Acceptance
- Test: an agent with limit 100 and usage 95 does not call the provider for a request estimated at 10.
- Test: the recorded cost equals tokens × price.
- Test: unknown pricing keeps `cost_micros=null` and `cost_status=unknown`, never `0`.
