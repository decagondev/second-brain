---
title: "ADR-0014: If the parent organisation reads company data, it does so as a separate tenant, never as a superuser"
summary: "If the parent organisation reads company data, it does so as a separate tenant, never as a superuser."
---

# ADR-0014: If the parent organisation reads company data, it does so as a separate tenant, never as a superuser

| Status | Date | Deciders |
|---|---|---|
| Proposed | 2026-11-18 | Platform lead, security lead |

## Context

The parent may want portfolio-wide insights. A superuser path would bypass every law and every company's trust.

## Decision

Any parent-org access to company data is modelled as its own tenant with explicit, per-company, audited grants approved by that company, plus legal review. There is no superuser role in any cell. Until this is decided (Q-01), v1 has no cross-company features.

## Alternatives rejected

| Option | Why not |
|---|---|
| Platform superuser | Breaks LAW-02 by design |

## Consequences

- Cross-company features are slower to build and require consent flows
- Owner: parent-org legal + platform lead. Decision needed before W6
