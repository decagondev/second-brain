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
| Status | Open |

## Why

One heavy user can spend a space's budget before anyone notices.

## Acceptance criteria

- [ ] `limits: { monthly_usd_per_person }`; the person is told at 80%
- [ ] At 100% their asks and research move to the economy tier and their routines pause; anything they review keeps working
- [ ] A steward can lift one person's limit for the month, and the lift is audited

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Setting
- [ ] Metering
- [ ] Lift
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
