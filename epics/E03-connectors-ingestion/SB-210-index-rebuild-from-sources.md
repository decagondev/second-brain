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
| Status | Open |

## Why

If the index can't be rebuilt, it has quietly become a system of record.

## Acceptance criteria

- [ ] Scheduled drill drops and rebuilds an index; the (doc, version, ACL) set is identical
- [ ] Rebuild time recorded per tenant

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Rebuild command
- [ ] Set-equality checker
- [ ] Drill schedule
- [ ] Ledger update
- [ ] Write or extend the law tests: LAW-13
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
