---
title: "SB-704: Root-key re-wrap and key custody transfer"
summary: "Custody of data is custody of keys."
---

# SB-704: Root-key re-wrap and key custody transfer

| Field | Value |
|---|---|
| Epic | [E08 Factory and Hand-off](README.md) |
| Wave / slice | W7 / S7.2 Hand-off |
| Stories | US-08.2 |
| Manual | [`sb-factory(7)`](../../manual/sb-factory.7.md) |
| Laws | [LAW-18](../../manual/sb-laws.7.md#law-18) |
| Depends on | [SB-703](../E08-factory-handoff/SB-703-placement-adapter-for-tenant-owned-cloud.md) |
| Status | Open |

## Why

Custody of data is custody of keys.

## Acceptance criteria

- [ ] Root key re-wrapped under tenant KMS before the factory wrap is destroyed
- [ ] Key-wrap inventory per cell exposed to `sb factory status`

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Re-wrap operation
- [ ] Wrap inventory
- [ ] Ordering test that fails if destroy precedes wrap
- [ ] Write or extend the law tests: LAW-18
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
