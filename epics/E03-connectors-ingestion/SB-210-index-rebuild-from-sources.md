---
title: "SB-210: Index rebuild from sources"
summary: "If the index can't be rebuilt, it has quietly become a system of record."
---

# SB-210: Index rebuild from sources

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.3 Pipeline |
| Stories | US-03.4 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | [LAW-13](../../manual/sb-laws.7.md#law-13) |
| Depends on | [SB-208](../E03-connectors-ingestion/SB-208-incremental-sync-via-changes-with-freshness.md) |
| Status | In review |

## Why

If the index can't be rebuilt, it has quietly become a system of record.

## Acceptance criteria

- [x] Scheduled drill drops and rebuilds an index; the (doc, version, ACL) set is identical (`ingest.Drill`, weekly)
- [ ] Rebuild time recorded per tenant

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Rebuild command
- [x] Set-equality checker
- [x] Drill schedule
- [x] Ledger update
- [x] Write or extend the law tests: LAW-13
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: `ingest.Drill` snapshots, drops, rebuilds from the change feed's start, and compares the `(cite, chunk id, ACL)` sets. It reports missing and extra entries by id, and the time taken. `DrillEvery` is weekly. A unit test plants a stray chunk and checks it is reported. `TestLAW13_RebuildDrill` runs the production drill over random corpora (it went green at once, since the rebuild path exists from SB-206; the existing LAW-13 self-test covers the check).

**Left:** "Rebuild time recorded per tenant" in a running cell. The report carries the time, and wiring it into the cell's metrics and audit comes with the cell runtime and SB-607.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
