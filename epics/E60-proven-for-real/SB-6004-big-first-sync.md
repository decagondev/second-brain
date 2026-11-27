---
title: "SB-6004: A big tenant's first sync"
summary: "A big tenant's first sync has never been tried: providers throttle, and a restart starts again from the top."
---

# SB-6004: A big tenant's first sync

| Field | Value |
|---|---|
| Epic | [E60 Proven for Real](README.md) |
| Wave / slice | W59 / S59.2 |
| Stories | US-60.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-05, LAW-13 |
| Depends on | SB-5901, SB-3201 |
| Status | Done |

## Why

A big tenant's first sync has never been tried: providers throttle, and a restart starts again from the top.

## Acceptance criteria

- [x] First syncs are paced by sb-cell(5) `sync.max_per_minute` (default 600) and wait on `Retry-After` and 429s instead of failing
- [x] A checkpoint every 500 items lets a restart resume where it stopped
- [x] Health shows each connector's progress and time left, and an answer from a space still in its first sync carries `notice: still_indexing`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Pacing and checkpoints in the ingest pipeline
- [x] Health progress and the notice
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `ingest.Pace` (sb-cell(5) `sync.max_per_minute`, default 600) spaces every read; a 429 waits its Retry-After (msgraph.Error carries it) or 1, 2, 4 seconds up to 5 minutes, 8 times, counted in `sb_ingest_throttled_total`. A first sync keeps a checkpoint at the start and every 500 changes under `CONNECTOR--first` and resumes from it; health shows `first_sync` with items, of and minutes left; answers carry `notice_indexing` (a field of its own, beside `notice`, so it never hides another notice) and the web UI says so. Tested in TestFirstSyncIsPacedPatientAndResumes and TestStillIndexingThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
