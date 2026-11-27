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
| Status | Open |

## Why

Microsoft sensitivity labels mean nothing to the brain.

## Acceptance criteria

- [ ] The connector reads each item's sensitivity label id; sb-cell(5) `labels.microsoft` maps ids to the four levels
- [ ] An unmapped label counts as `restricted` until mapped; health lists unmapped labels by id and count
- [ ] Changing a mapping relabels at the next sync

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Label ids from Graph
- [ ] Mapping and the restricted default
- [ ] Health line
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
