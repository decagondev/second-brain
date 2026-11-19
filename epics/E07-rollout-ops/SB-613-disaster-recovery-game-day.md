---
title: "SB-613: Disaster-recovery game day"
summary: "Restore a cell from backups inside its region, on the clock."
---

# SB-613: Disaster-recovery game day

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.4 Harden |
| Stories | US-07.6 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | [LAW-11](../../manual/sb-laws.7.md#law-11), [LAW-13](../../manual/sb-laws.7.md#law-13) |
| Depends on | [SB-412](../E05-actions-approvals-audit/SB-412-audit-and-approval-backups-3-2-1-and-restore.md), [SB-210](../E03-connectors-ingestion/SB-210-index-rebuild-from-sources.md) |
| Status | Open |

## Why

Restore a cell from backups inside its region, on the clock.

## Acceptance criteria

- [ ] Cell restored within RTO 4 hours, RPO 15 minutes
- [ ] Audit chain verifies after restore

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Plan
- [ ] Run
- [ ] Retro and fixes
- [ ] Write or extend the law tests: LAW-11, LAW-13
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
