---
title: "SB-904: Every schedule running in the cell"
summary: "Sync, retention, reports, access checks and anchors exist as functions; nothing runs them yet."
---

# SB-904: Every schedule running in the cell

| Field | Value |
|---|---|
| Epic | [E09 Cells on GCP](README.md) |
| Wave / slice | W9 / S9.2 Runtime jobs |
| Stories | US-09.2 |
| Manual | `sb-runtime(7)` (new), [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | [LAW-11](../../manual/sb-laws.7.md#law-11), [LAW-13](../../manual/sb-laws.7.md#law-13), [LAW-17](../../manual/sb-laws.7.md#law-17) |
| Depends on | [SB-901](SB-901-cloud-run-services-per-cell.md) |
| Status | In review |

## Why

Sync, retention, reports, access checks and anchors exist as functions; nothing runs them yet.

## Acceptance criteria

- [ ] Cloud Scheduler and Pub/Sub run incremental sync (SB-208), nightly retention (SB-611), the weekly report (SB-606), the monthly LAW-17 check (SB-706) and the hourly audit anchor (scheduled against the fake; the `jobs` runner doing the work in a running cell is SB-903)
- [x] Anchors go to a Cloud Storage bucket with a locked retention policy (ADR-0012)
- [x] A missed run raises an alert, never a silent skip

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Scheduler adapter
- [x] WORM anchor adapter
- [x] Missed-run alert
- [ ] Close the schedule items on SB-208, SB-606, SB-611, SB-706 (when the schedule runs in a real cell, SB-903)
- [x] Write or extend the law tests: LAW-11, LAW-13, LAW-17
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S9.2a): sb-runtime(7) gained "Schedules, exactly", with six jobs, their times, laws, the watchdog and the WORM bucket. sb-ingest(7) and sb-observability(7) follow it. `internal/core/schedule` defines the jobs once, with the watchdog `Watch`. It marks a job overdue past its grace, or when its last run is missing or unreadable, sets `sb_job_overdue{job}`, and alerts Sev-1 for a job that carries a law and Sev-2 otherwise. `gcp.Scheduler` makes the Cloud Scheduler jobs, publishing to the cell's topic, and corrects one that drifted. `gcp.Bucket` is a `backup.Target` over Cloud Storage. It refuses a bucket whose retention isn't locked or is under seven years, and writes with `ifGenerationMatch=0` so nothing is overwritten. The GCP fake now has Cloud Scheduler and Cloud Storage with retention, locks and refused deletes. Four new law tests: `TestLAW11_AnchorsAreWriteOnce` (overwrite and delete attempts fail), and the missed-run checks for LAW-11, LAW-13 and LAW-17. Self-tests catch a watchdog that treats unknown as fine, and one that alerts law jobs below Sev-1.

**Left:** the `jobs` runner doing the work in a running cell, and closing the schedule items on SB-208, SB-606, SB-611 and SB-706 (SB-903).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
