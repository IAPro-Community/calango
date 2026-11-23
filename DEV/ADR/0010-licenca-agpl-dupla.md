# ADR 0010 — Open source under AGPL-3.0, with commercial license, CLA and trademark

Status: accepted · 2026-11-23 · publication under `IAPro-Community`

## Context
The project will be built with the community, but it is a business: the owner leads and the revenue comes from services and commercial licenses. A permissive license would let third parties host a competing copy without contributing back. The project also depends on third-party contributions, which requires assigning rights to keep the commercial license viable.

## Decision
1. **Code license:** GNU AGPL-3.0 (`LICENSE`).
2. **Commercial license** for those who cannot or do not want to comply with the AGPL (`COMMERCIAL.md`).
3. **CLA** required for contributions (`CLA.md`): copyright and patent licenses, with the right to relicense.
4. **Trademark** protected separately (`TRADEMARKS.md`).
5. **Paid services outside the repository:** relay, push, managed VPS, model credits, billing and licensing stay in services operated by Calango. The open binary does not contain billing logic as an access barrier.
6. **Dependencies:** the Orquestrador Maestro (Apache-2.0) is consumed via npm. Apache-2.0 is compatible with AGPL-3.0 for this use.

## Discarded alternatives
- **Apache-2.0 / MIT:** would allow a competing hosted fork without any obligation to publish.
- **GPL-3.0:** does not cover use as a network service.
- **Source-available (BSL, SSPL, FSL):** not open source licenses approved by the OSI; the owner asked for open source.

## Accepted risks
- Companies that avoid AGPL may prefer the commercial license. That is revenue, not only cost.
- The VPS license check can be removed by whoever compiles the code. That is why what is charged stays outside the binary (item 5).
- The CLA requires legal review before publication.

## Open items
- Legal review of `CLA.md` and `COMMERCIAL.md`.
- Real contact address, terms of service and privacy policy.
- CLA signature mechanism in the PR (bot or checkbox).
