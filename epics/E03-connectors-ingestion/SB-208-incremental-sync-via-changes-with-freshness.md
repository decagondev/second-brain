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
| Status | In review |

## Why

Answers are only useful if they reflect current documents.

## Acceptance criteria

- [ ] p95 change-to-index time is 15 minutes or less in staging
- [x] Cursor stored per connector; restart resumes

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Change cursor store
- [x] Scheduler
- [x] Freshness metric
- [ ] Staging load test
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19:
- `ports.CursorStore`, with a file adapter (`internal/adapters/cursor/filestore`) for development; cells keep cursors in their database.
- `ingest.Syncer.Once` stores the new cursor only after the whole sync succeeded, and `Run` syncs every 60 s plus jitter.
- `internal/obs` gains a Prometheus text registry (`Metrics`, served at `/metrics`) and `Freshness`, which exports `sb_ingest_last_success_age_seconds` and `sb_ingest_sync_failures_total` per tenant and connector.
- A test fails the second fetch, checks the cursor didn't move, then resumes and checks both documents are indexed.

**Left:** "p95 change-to-index time is 15 minutes or less in staging" and the staging load test need a staging cell, which needs a cloud adapter (ADR-0018). Locally a sync takes milliseconds.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
