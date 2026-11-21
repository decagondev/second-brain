---
title: "SB-1503: The ingest service: syncs that run"
summary: "Connectors and the pipeline exist; nothing runs a sync on a schedule and keeps its cursor."
---

# SB-1503: The `ingest` service: syncs that run

| Field | Value |
|---|---|
| Epic | [E15 Cell Runtime](README.md) |
| Wave / slice | W14 / S14.2 |
| Stories | US-15.1 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-07, LAW-13 |
| Depends on | [SB-1501](SB-1501-cell-configuration-and-wiring.md) |
| Status | Done |

## Why

Connectors and the pipeline exist; nothing runs a sync on a schedule and keeps its cursor.

## Acceptance criteria

- [x] `cmd/sb-ingest` runs each connector's sync with its cursor stored durably, re-routing first, with the secret and field hooks counted in `/metrics`
- [x] A failing connector backs off and alerts without stopping the others; a lost cursor resyncs from the start rather than skipping
- [x] A law test drives a local cell through ingest, a scope change and a source deletion

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Sync runner and cursor store
- [x] Backoff and alerts
- [x] Law test
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-21 (slice S14.2): `internal/cell/ingestsvc` builds each connector's pipeline from the cell (`wire.Cell`): the cell's index and embedder, the spaces registry for routing, and counters for dropped secrets, fields and reroutes in `/metrics`. `Run` re-routes every connector first, then syncs each in its own loop from its cursor in `state/cursors`; a failure backs that connector off (one minute doubling to 15) and never slows another, and a connector stale past 15 minutes sends one `ingest_stale` alert until it recovers. `cmd/sb-ingest` runs it on its own once the stores are durable (SB-1508); until then `sb-api --with ingest` runs it in the api's process (sb-cell(5)). `tests/cell` drives a running cell through a recorded Drive fake: `TestLAW13_IngestThroughARunningCell` (ingest, then ann asks and is answered; the source stops sharing with her and she isn't; the file is deleted and its chunks leave the index), `TestIngestRestartResumes` (a lost cursor resyncs everything) and `TestFailingConnectorAlertsAndBacksOff`. Drive's classification label is wired from `vault://tenants/TENANT/connectors/docs-classification`, now an allowed vault path.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
