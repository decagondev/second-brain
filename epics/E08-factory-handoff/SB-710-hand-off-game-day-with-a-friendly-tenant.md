---
title: "SB-710: Hand-off game day with a friendly tenant"
summary: "Rehearse it before anyone needs it."
---

# SB-710: Hand-off game day with a friendly tenant

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.3 Exit paths |
| Stories | US-08.5 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-17](../../manual/sb-laws.7.md#law-17), [LAW-18](../../manual/sb-laws.7.md#law-18) |
| Depends on | [SB-706](../E08-factory-handoff/SB-706-access-path-enumeration-and-law-17-check.md), [SB-707](../E08-factory-handoff/SB-707-releases-after-hand-off.md), [SB-708](../E08-factory-handoff/SB-708-detach-and-export-package.md) |
| Status | Open |

## Why

Rehearse it before anyone needs it.

## Acceptance criteria

- [ ] A friendly tenant's cell handed off end to end; laws pass; audit chain verifies across the move
- [ ] Detach rehearsed on a copy

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Plan
- [ ] Run
- [ ] Retro, bugs and runbook fixes
- [ ] Write or extend the law tests: LAW-17, LAW-18
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
