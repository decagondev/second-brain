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
| Status | Open |

## Why

Connectors and the pipeline exist; nothing runs a sync on a schedule and keeps its cursor.

## Acceptance criteria

- [ ] `cmd/sb-ingest` runs each connector's sync with its cursor stored durably, re-routing first, with the secret and field hooks counted in `/metrics`
- [ ] A failing connector backs off and alerts without stopping the others; a lost cursor resyncs from the start rather than skipping
- [ ] A law test drives a local cell through ingest, a scope change and a source deletion

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Sync runner and cursor store
- [ ] Backoff and alerts
- [ ] Law test
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
