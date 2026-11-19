---
title: "SB-706: Access-path enumeration and LAW-17 check"
summary: "Prove the factory kept only what was agreed."
---

# SB-706: Access-path enumeration and LAW-17 check

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.2 Hand-off |
| Stories | US-08.3 |
| Manual | [`sb-tenant(1)`](../../manual/sb-tenant.1.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | [LAW-17](../../manual/sb-laws.7.md#law-17) |
| Depends on | [SB-705](../E08-factory-handoff/SB-705-hand-off-workflow-with-dual-approval.md) |
| Status | Open |

## Why

Prove the factory kept only what was agreed.

## Acceptance criteria

- [ ] `sb tenant verify --law LAW-17` enumerates factory access paths from IaC state and IAM and fails on any not in the hand-off record
- [ ] Monthly re-check alerts on drift

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Enumerator
- [ ] Comparison with hand-off record
- [ ] Monthly job
- [ ] Alert
- [ ] Write or extend the law tests: LAW-17
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
