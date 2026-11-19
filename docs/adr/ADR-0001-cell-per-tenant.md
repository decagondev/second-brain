---
title: "ADR-0001: One cell per tenant, stamped from one template"
summary: "One cell per tenant, stamped from one template."
---

# ADR-0001: One cell per tenant, stamped from one template

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

Acquired companies won't accept their data sharing an index or database with other companies, and a single missed `WHERE tenant =` in a shared store would be a cross-company leak.

## Decision

Each tenant gets its own database, index schema, vault namespace, KMS key, queue and region pin. All cells are stamped from one infrastructure template and run the same release. The shared control plane holds no tenant content: only routing, releases, kill flags and attestations.

## Alternatives rejected

| Option | Why not |
|---|---|
| Shared multi-tenant store with row-level security | One policy bug leaks across companies; hard to prove isolation to an acquired company's security team |
| Per-company custom deployments | Forks and drift; no single release train |

## Consequences

- Isolation is structural; LAW-02 becomes easy to test and to explain
- Higher fixed cost per tenant (one small database each). Acceptable at portfolio scale (tens of companies, not thousands)
- Cross-company features need an explicit, separate design (ADR-0014)
