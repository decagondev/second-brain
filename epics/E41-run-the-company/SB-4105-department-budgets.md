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
| Status | Done |

## Why

Spend is known per tenant, not per department.

## Acceptance criteria

- [x] A space can set `budget_usd_month`; asks and scheduled runs are metered by space and rolled up the tree, with the forecast
- [x] Stewards are told at 80%; at 100% scheduled briefs and updates pause until the month ends
- [x] Asks carry on up to the tenant's budget; the admin console shows each branch's spend

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Metering
- [x] Notify
- [x] Admin console
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. Space budget_usd_month (at most the tenant's and the department's); ports.ModelRequest.Space, budget.SpaceMeter and Guard.RecordIn meter each call by space (localfs space_usd); wire branchSpend rolls teams up; spaceOver pauses briefs and updates (BriefSpaceBudget); notify SpaceBudgets warns stewards once at 80% and once at 100%; GET /v1/admin/costs/spaces and web AdminSpend. Asks carry on. Tests: TestRecordInMetersTheSpace, TestMeteredBySpace, TestSpendBySpace, TestSpaceBudgetWarnings, TestRequestsAndBudgetsThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
