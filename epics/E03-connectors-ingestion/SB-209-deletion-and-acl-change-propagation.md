---
title: "SB-209: Deletion and ACL-change propagation"
summary: "A deleted or restricted document must stop being retrievable promptly."
---

# SB-209: Deletion and ACL-change propagation

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.3 Pipeline |
| Stories | US-03.2 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01) |
| Depends on | [SB-208](../E03-connectors-ingestion/SB-208-incremental-sync-via-changes-with-freshness.md) |
| Status | Open |

## Why

A deleted or restricted document must stop being retrievable promptly.

## Acceptance criteria

- [ ] Deletes and ACL changes reach the prefilter within the freshness SLA
- [ ] Live check still refuses in the gap (LAW-01 test covers stale prefilter)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Tombstones
- [ ] ACL delta handling
- [ ] Stale-prefilter law test
- [ ] Write or extend the law tests: LAW-01
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
