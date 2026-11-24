---
title: "SB-3505: Month-end forecast"
summary: "The cost statement shows the month so far, not where it is heading."
---

# SB-3505: Month-end forecast

| Field | Value |
|---|---|
| Epic | [E35 Trust and Control](README.md) |
| Wave / slice | W34 / S34.2 |
| Stories | US-35.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | LAW-03 |
| Depends on | SB-709 |
| Status | Done |

## Why

The cost statement shows the month so far, not where it is heading.

## Acceptance criteria

- [x] `GET /v1/admin/costs` adds `forecast_usd` from the last 7 days' rate
- [x] Fewer than 3 days of data gives no number
- [x] A forecast over the budget fires budget_warning early, once a month

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Costs
- [x] Alert
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S34.2 with tests; make check green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
