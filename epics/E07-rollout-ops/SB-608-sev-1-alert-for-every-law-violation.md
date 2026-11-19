---
title: "SB-608: Sev-1 alert for every law violation"
summary: "A law that can break silently is not a law."
---

# SB-608: Sev-1 alert for every law violation

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.3 Operate |
| Stories | US-07.4 |
| Manual | [`sb-laws(7)`](../../manual/sb-laws.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-06](../../manual/sb-laws.7.md#law-06), [LAW-07](../../manual/sb-laws.7.md#law-07), [LAW-11](../../manual/sb-laws.7.md#law-11), [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-607](../E07-rollout-ops/SB-607-per-tenant-observability.md) |
| Status | Open |

## Why

A law that can break silently is not a law.

## Acceptance criteria

- [ ] Every law with a production check has an alert and a runbook
- [ ] Alert test fires each one in staging

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Alert rules
- [ ] Runbooks
- [ ] Fire drill
- [ ] Write or extend the law tests: LAW-01, LAW-02, LAW-06, LAW-07, LAW-11, LAW-16
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
