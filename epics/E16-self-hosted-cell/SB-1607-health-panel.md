---
title: "SB-1607: Cell health in the console"
summary: "Operators should see missed jobs and stale connectors without reading metrics."
---

# SB-1607: Cell health in the console

| Field | Value |
|---|---|
| Epic | [E16 Self-hosted Cell](README.md) |
| Wave / slice | W15 / S15.4 |
| Stories | US-16.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-13, LAW-16 |
| Depends on | — |
| Status | Done |

## Why

Operators should see missed jobs and stale connectors without reading metrics.

## Acceptance criteria

- [x] `GET /v1/admin/health` lists each job's last finish and overdue state and each connector's last success and failures, from the cell's own records
- [x] The console shows it first, overdue and stale items at the top
- [x] UI and API tests

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Endpoint
- [x] Console panel
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S15.4): `internal/core/health` builds the cell's health from its own records: each scheduled job's last finish and overdue state, decided exactly as the watchdog decides it (an unreadable or missing record is overdue), and each connector's last success, last failure and failures since, stale past the 15-minute freshness target or with no success at all; overdue and stale items first. The ingest service now writes each connector's sync record (`runsfs.Syncs`, `state/runs/sync-KIND.json`) beside the job records, so every service reads the same. `GET /v1/admin/health` (operators) answers it, and an unreadable connector record is `EUNAVAIL`, never a healthy report (`TestLAW03_HealthNeverShowsAnUnreadableRecordAsHealthy`). The console opens on it. Tests: the core report, the sync records, a running cell through a failing source (`TestHealthFromTheCellsRecords`), and the console with axe. The self-hosted demo's smoke checks read it as the demo's operator.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
