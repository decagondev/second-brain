---
title: "SB-4905: When a source changes"
summary: "A note or brief an agent drafted goes stale the day its source changes."
---

# SB-4905: When a source changes

| Field | Value |
|---|---|
| Epic | [E49 Agents That Check With You](README.md) |
| Wave / slice | W48 / S48.2 |
| Stories | US-49.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-01, LAW-14, LAW-26 |
| Depends on | SB-4006, SB-4103 |
| Status | Done |

## Why

A note or brief an agent drafted goes stale the day its source changes.

## Acceptance criteria

- [x] `on: source-changed` redrafts what an agent drafted from a changed document or policy in the last 90 days, with the change highlighted
- [x] Changes are batched to at most one redraft a day, and nothing republishes without the approver
- [x] A source the approver can no longer open drops the redraft for them

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Trigger
- [x] Dependency lookup from traces
- [x] Batching
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. sb-space(5) `on: [source-changed]` (`manifest.Space.On`, `Watches`). `internal/core/ask/sourcechanged.go`: the notify job passes each document's newest version first seen in the run's window; `NoteSourceChanges` marks it pending on every agent draft of a watching space from the last 90 days that cited it or was narrowed to it and is older than the version. `RedraftChanged` then redrafts each draft with pending changes once a UTC day at most, as its approver (the steward who published it, or the one it ran as), from the same sources only, into a new draft "Source changed: TITLE" with the earlier text in `versions` (so the queue shows the diff) and `changed`; the earlier draft hands watching on to it (`Next`). An approver who is no longer a steward or can't open a changed source drops it (`source_dropped`); a paused, killed or capped agent waits. Nothing is republished. A policy is a document, so its new version counts. Tested in TestScenariosSourcesAndStartersThroughARunningCell and TestSpaceOnSourceChanged.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
