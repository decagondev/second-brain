---
title: "ADR-0006: Buy commodity infrastructure; build the trust layer"
summary: "Buy commodity infrastructure; build the trust layer."
---

# ADR-0006: Buy commodity infrastructure; build the trust layer

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

A small team can't run databases, queues and SSO well and also build a sound permission layer.

## Decision

Buy: the model API, a managed container runtime, managed Postgres with pgvector, a managed queue, managed vault/KMS, SSO via the company's own IdP, and existing connector SDKs. Build: policy, retrieval authorisation, citations, audit, approvals, kill switches, evals, rollout and the manifest/CLI.

## Alternatives rejected

| Option | Why not |
|---|---|
| Buy an end-to-end enterprise search product | Can't prove LAW-01 to LAW-05 inside someone else's black box; lock-in across a portfolio |
| Build everything | Slow, and most of it isn't differentiating |

## Consequences

- The team's effort goes into the guarantees
- Vendor adapters must stay behind ports (P6) so each can be swapped
