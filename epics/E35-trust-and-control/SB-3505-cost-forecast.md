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
| Status | Open |

## Why

The cost statement shows the month so far, not where it is heading.

## Acceptance criteria

- [ ] `GET /v1/admin/costs` adds `forecast_usd` from the last 7 days' rate
- [ ] Fewer than 3 days of data gives no number
- [ ] A forecast over the budget fires budget_warning early, once a month

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Costs
- [ ] Alert
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
