---
title: "SB-4105: Department budgets"
summary: "Spend is known per tenant, not per department."
---

# SB-4105: Department budgets

| Field | Value |
|---|---|
| Epic | [E41 Run the Company](README.md) |
| Wave / slice | W40 / S40.2 |
| Stories | US-41.2 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07 |
| Depends on | SB-3505 |
| Status | Open |

## Why

Spend is known per tenant, not per department.

## Acceptance criteria

- [ ] A space can set `budget_usd_month`; asks and scheduled runs are metered by space and rolled up the tree, with the forecast
- [ ] Stewards are told at 80%; at 100% scheduled briefs and updates pause until the month ends
- [ ] Asks carry on up to the tenant's budget; the admin console shows each branch's spend

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Metering
- [ ] Notify
- [ ] Admin console
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
