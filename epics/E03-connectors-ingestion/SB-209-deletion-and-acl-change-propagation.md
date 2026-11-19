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
| Status | In review |

## Why

A deleted or restricted document must stop being retrievable promptly.

## Acceptance criteria

- [ ] Deletes and ACL changes reach the prefilter within the freshness SLA
- [x] Live check still refuses in the gap (LAW-01 test covers stale prefilter)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Tombstones: the source's change feed is the tombstone record; a delete removes every chunk of the document
- [x] ACL delta handling
- [x] Stale-prefilter law test
- [x] Write or extend the law tests: LAW-01
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19: `TestLAW01_PrefilterCatchesUp` was red first, then green. It shares, re-shares and deletes documents at random. In the gap before sync, the connector's live verdict must follow the source exactly; after one sync, every indexed prefilter must equal the source's viewers and deleted documents must be gone. The mechanism landed in SB-206 (deletes remove chunks; permission-only changes force a same-version re-ingest), so this ticket is the law, the manual wording, and a self-test that catches a pipeline ignoring permission-only changes.

**Left:** "within the freshness SLA" in staging (with SB-208's staging run).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
