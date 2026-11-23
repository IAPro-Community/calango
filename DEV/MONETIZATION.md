# Business model — Calango (freemium)

Status: initial proposal. Prices are hypotheses to validate, not decisions.

Update: the hybrid model (free PC node, paid VPS node, tunnel via relay) is in [PRODUCT_HYBRID.md](PRODUCT_HYBRID.md). This plan table predates that document and should be read together with it.

## Principle

The cost that weighs most is the model (tokens) and cloud execution. The free plan must be useful without being expensive for us: so the free plan runs locally and uses the user's own key or subscription (BYOK).

## Plans (proposal)

| | Free | Pro | Team | Business |
|---|---|---|---|---|
| Who | to try it | a person who uses it every day | a small team | a company |
| Local machines | 1 | unlimited | unlimited | unlimited |
| Cloud (24/7 runner) | no | 1 runner | runners per person | dedicated runners |
| Routines | 2 | unlimited | unlimited | unlimited |
| Channels | 1, only you | unlimited | shared channels | channels + role permissions |
| Projects in ADE | 1 | unlimited | unlimited | unlimited |
| Agent browser | basic, no persistent profile | persistent profile | profile per person | profile per person + audit |
| Approvals | only you | per device (push) | per person | flow with owner and substitute |
| Model credits | BYOK | included + extra purchase | included per seat | contract |
| Support | community | email | priority | SLA |
| SSO, audit, retention | no | no | optional | yes |

## Monetization beyond the subscription

1. **Usage credits**: tokens of models hosted by us, with a markup over cost. One-off purchase when the plan runs out.
2. **Runner minutes**: 24/7 cloud execution billed by usage above the quota.
3. **Storage**: files, memory and history above the quota.
4. **Premium connectors**: third-party integrations with a per-call cost.
5. **Marketplace (future)**: ready-made routines and agents sold by creators, with a commission.

## How to measure (basis of the plan)

Every call goes through a model gateway that records tokens, cost and owner (see [ARCHITECTURE.md](ARCHITECTURE.md)). The same record feeds billing, plan limits and the usage dashboard. Without reliable measurement, there is no viable freemium.

## Limits and controls

- Plan quota enforced in the core (`Entitlements` per owner), not in the interface.
- When exceeded: the agent stops and asks the owner to act; it never charges silently.
- Spending limit per agent and per routine, always visible.

## Payment

- **Web and desktop**: Stripe (subscriptions and credits).
- **iOS and Android**: App Store and Google Play require their own purchase system for digital subscriptions consumed inside the app, with their commission (15 to 30%). Confirm the current rules before setting the in-app price. Alternative: offer purchase only on the web and read the subscription in the app (rules change by store and by country; check).

## Risks

- A free plan with an expensive model becomes a loss: BYOK on the free plan solves this.
- Store commissions erode mobile margin: push purchase to the web where the policy allows it.
- Reference prices (ChatGPT, Grok, Claude) create expectations of low prices; the differentiator must be cloud + ADE + control, not "one more chat".
- Dot was shut down (see WORKLOG): a companion subscription without daily usefulness did not hold up.

## Initial metrics

Free→paid conversion, weekly retention, model cost per active user, revenue per user, churn, active routines per account.

## Next step

Validate with 5 to 10 people from the target audience: which free-plan limit would make them pay, and what they would pay for cloud compared with what they already pay in AI subscriptions.
