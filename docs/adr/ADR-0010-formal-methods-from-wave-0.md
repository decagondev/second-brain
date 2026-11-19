---
title: "ADR-0010: TLA+ and Alloy from Wave 0; laws as the main gate from Wave 3"
summary: "TLA+ and Alloy from Wave 0; laws as the main gate from Wave 3."
---

# ADR-0010: TLA+ and Alloy from Wave 0; laws as the main gate from Wave 3

| Status | Date | Deciders |
|---|---|---|
| Accepted | 2026-11-18 | Platform lead, security lead |

## Context

Approvals, rollout and permissions are small state machines and relations where concurrency bugs and missed cases cause real harm. They're cheap to model before code exists.

## Decision

Model approvals and kill switches (`Approvals.tla`) and rollout (`Rollout.tla`) in TLA+, and permissions and tenancy (`Permissions.als`) in Alloy, in Wave 0, checked in CI. Each law is tracked in the assurance ledger from Stated to Modelled, Tested, Enforced and Watched. Unit, regression and smoke tests stay forever; from Wave 3, laws, models and evals are the main release gate.

## Alternatives rejected

| Option | Why not |
|---|---|
| Tests only | Tests find bugs in what was built, not in what was designed |
| Full verification of the Go code | Cost far beyond the benefit for v1 |

## Consequences

- Design flaws surface in W0 (e.g. the promote guard for LAW-08)
- Specs must be kept in step with code; state names are shared to make drift visible
