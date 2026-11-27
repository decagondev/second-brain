---
title: "SB-5902: Purview labels"
summary: "Microsoft sensitivity labels mean nothing to the brain."
---

# SB-5902: Purview labels

| Field | Value |
|---|---|
| Epic | [E59 Reach Microsoft 365](README.md) |
| Wave / slice | W58 / S58.1 |
| Stories | US-59.1 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-03, LAW-32 |
| Depends on | SB-5901, SB-5405 |
| Status | Done |

## Why

Microsoft sensitivity labels mean nothing to the brain.

## Acceptance criteria

- [x] The connector reads each item's sensitivity label id; sb-cell(5) `labels.microsoft` maps ids to the four levels
- [x] An unmapped label counts as `restricted` until mapped; health lists unmapped labels by id and count
- [x] Changing a mapping relabels at the next sync

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Label ids from Graph
- [x] Mapping and the restricted default
- [x] Health line
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `SharePoint.label` maps `sensitivityLabel.labelId` by sb-cell(5) `labels.microsoft`; an unmapped id is `restricted` and counted (`SharePoint.Unmapped`), and the health report's `unmapped_labels` lists ids and counts only; a new mapping relabels at the next fetch. Tested in TestSharePointSync and TestMicrosoft365ThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
