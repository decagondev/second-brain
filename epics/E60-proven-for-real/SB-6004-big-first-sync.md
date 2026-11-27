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
| Status | Open |

## Why

A big tenant's first sync has never been tried: providers throttle, and a restart starts again from the top.

## Acceptance criteria

- [ ] First syncs are paced by sb-cell(5) `sync.max_per_minute` (default 600) and wait on `Retry-After` and 429s instead of failing
- [ ] A checkpoint every 500 items lets a restart resume where it stopped
- [ ] Health shows each connector's progress and time left, and an answer from a space still in its first sync carries `notice: still_indexing`

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Pacing and checkpoints in the ingest pipeline
- [ ] Health progress and the notice
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
