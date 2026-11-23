# ADR 0006 — Original code; references for reading only

Status: accepted · 2026-11-23

## Context
The product is inspired by Tucano (Apache-2.0) and agentes.chat (AGPL-3.0, with a registered trademark and a CLA).

## Decision
No code, text, test, prompt or asset from the reference projects enters this repository. They serve only to understand the product and its problems (issues, changelog, ADRs). The specs describe behavior in our own words.

## Exception
The Orquestrador Maestro (Apache-2.0) is consumed from the official npm package, per [ADR 0009](0009-orquestrador-maestro.md). It is not a code copy.

## Consequences
- Full license freedom for Calango.
- More implementation work; accepted.
- PR review must reject fragments that look copied.
