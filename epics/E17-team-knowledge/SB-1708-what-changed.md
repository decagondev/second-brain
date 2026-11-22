---
title: "SB-1708: What changed in my spaces"
summary: "Ingest knows every new version, and nobody is told what changed in the things they work on."
---

# SB-1708: What changed in my spaces

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.4 |
| Stories | US-17.4 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-01, LAW-13 |
| Depends on | — |
| Status | Done |

## Why

Ingest knows every new version, and nobody is told what changed in the things they work on.

## Acceptance criteria

- [x] The index records when it first saw each version; versions written by a rebuild are marked as such
- [x] `GET /v1/changes?since=7d[&space=…]` lists documents with a new version in the window, in the person's spaces, each checked live, newest first
- [x] The UI shows What changed this week on the empty answer page; the weekly report counts changes per space
- [x] A rebuild never shows as a week of changes

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] First-seen column and rebuild mark
- [x] Endpoint and UI
- [x] Tests, including a rebuild
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S16.4): chunks carry `Rebuilt` beside `Seen`. A rebuild reads every version's first-seen time before it drops the index and gives it back on replay; a version it didn't know, or every version when the old index can't be read, gets the rebuild's time and is marked rebuilt. `ports.DocQuery.SeenSince` finds versions first seen since a time and never a rebuilt one, in both indexes. `GET /v1/changes?since=Nd` (1 to 90 days) lists up to 50 documents newest first: in a space what it holds, company-wide what the person's own spaces hold, each checked with `retrieve.Allowed`. The web UI shows What changed this week on the empty answer page; the weekly report counts changed documents per space. Tests: `TestLAW13_ARebuildIsNeverAWeekOfChanges` (drill rebuilds and lost indexes; its self-test catches a rebuild that forgets first-seen times), and a running cell (`TestWhatChangedThroughARunningCell`, the drill keeps every first-seen time).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
