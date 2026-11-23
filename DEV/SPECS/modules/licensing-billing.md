# Module: licensing-billing

Phase 3 · `cmd/calango-cloud` (services `license`, `billing`) + `internal/entitlements` on the node

## Goal
Charge for the plans and unlock paid features without making the node depend on the internet.

## License
- The Cloud issues an Ed25519-signed token: `{ account_id, plan, features[], nodes_max, exp }`, valid for 7 days, renewed automatically.
- The node validates the signature with the public key embedded in the binary. Without network, it is valid until `exp` + a 7-day tolerance.
- When it expires: the node **downgrades to Free** (paid features turn off, warnings appear). It never stops starting and never deletes data.

## Entitlements (in the core)
```go
type Entitlements struct {
    Plan        string
    VPSNodes    int
    Routines    int  // -1 = unlimited
    Projects    int
    RemoteScreen bool
    Push        bool
}
```
Checked in the service, not in the interface.

## Billing
- Stripe for web and desktop (subscription + credits).
- App Store and Google Play for purchases on the phone (via RevenueCat or direct integration).
- One source of truth for the plan in the Cloud, fed by webhooks from all three sources.

## Acceptance
- A VPS node offline for 10 days with a 7-day license keeps starting in Free mode and warns.
- A tampered token is refused.
