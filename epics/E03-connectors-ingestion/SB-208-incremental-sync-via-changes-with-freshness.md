---
title: "SB-208: Incremental sync via `changes` with freshness SLA"
summary: "Answers are only useful if they reflect current documents."
---

# SB-208: Incremental sync via `changes` with freshness SLA

| Field | Value |
|---|---|
| Epic | [E03 Connectors and Ingestion](README.md) |
| Wave / slice | W2 / S2.3 Pipeline |
| Stories | US-03.2 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | — |
| Depends on | [SB-206](../E03-connectors-ingestion/SB-206-ingest-pipeline-chunk-embed-tag-provenance.md) |
| Status | Open |

## Why

Answers are only useful if they reflect current documents.

## Acceptance criteria

- [ ] p95 change-to-index time is 15 minutes or less in staging
- [ ] Cursor stored per connector; restart resumes

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Change cursor store
- [ ] Scheduler
- [ ] Freshness metric
- [ ] Staging load test
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
