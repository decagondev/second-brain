---
title: "SB-5003: Personal limits"
summary: "One heavy user can spend a space's budget before anyone notices."
---

# SB-5003: Personal limits

| Field | Value |
|---|---|
| Epic | [E50 Run AI Like a Business](README.md) |
| Wave / slice | W49 / S49.1 |
| Stories | US-50.1 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4105 |
| Status | Done |

## Why

One heavy user can spend a space's budget before anyone notices.

## Acceptance criteria

- [x] `limits: { monthly_usd_per_person }`; the person is told at 80%
- [x] At 100% their asks and research move to the economy tier and their routines pause; anything they review keeps working
- [x] A steward can lift one person's limit for the month, and the lift is audited

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Setting
- [x] Metering
- [x] Lift
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: `limits: { monthly_usd_per_person }` in sb-tenant(5). The gateway meters each call for its person too (`budget.PersonMeter`, `person_usd` in the usage ledger). At 80% the notify job tells the person once a month (kind `limit`); at 100% their own asks, consultations and research use the small tier and their routines are skipped, while briefs and rhythm steps others review keep their tier. `POST /v1/spaces/NAME/limits/lift` lets a steward lift one member's limit for the month, audited before the lift. Tested in TestRoutingPoliciesAndLimits and TestSpendWithPurposeThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
