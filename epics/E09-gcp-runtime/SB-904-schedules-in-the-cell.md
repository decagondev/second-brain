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
| Status | Open |

## Why

Sync, retention, reports, access checks and anchors exist as functions; nothing runs them yet.

## Acceptance criteria

- [ ] Cloud Scheduler and Pub/Sub run incremental sync (SB-208), nightly retention (SB-611), the weekly report (SB-606), the monthly LAW-17 check (SB-706) and the hourly audit anchor
- [ ] Anchors go to a Cloud Storage bucket with a locked retention policy (ADR-0012)
- [ ] A missed run raises an alert, never a silent skip

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Scheduler adapter
- [ ] WORM anchor adapter
- [ ] Missed-run alert
- [ ] Close the schedule items on SB-208, SB-606, SB-611, SB-706
- [ ] Write or extend the law tests: LAW-11, LAW-13, LAW-17
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
