---
title: "SB-4804: Project close-out"
summary: "What a project learned leaves with the people who ran it."
---

# SB-4804: Project close-out

| Field | Value |
|---|---|
| Epic | [E48 Ways of Working](README.md) |
| Wave / slice | W47 / S47.1 |
| Stories | US-48.1 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-14, LAW-16 |
| Depends on | SB-4803, SB-4505 |
| Status | Done |

## Why

What a project learned leaves with the people who ran it.

## Acceptance criteria

- [x] **Close project** drafts a retrospective (goals, decisions, open waits, lessons) for the lead, every line cited
- [x] A lesson becomes a note only when a steward publishes it, and the next kickoff brief finds it
- [x] A closed collection is archived read-only; nothing is deleted

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Retrospective agent and eval suite
- [x] Lessons as notes
- [x] Archive
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `POST /v1/projects/ID/close` (lead only) sets the collection `closed`, after which it can't be changed or deleted, stops `kickoff`, puts each lesson in the space's queue as a note draft with its cite, and starts `close-out`, whose day-0 step is the retrospective: goals in the collection met or not, the decision log, waits open on its documents and the lessons, each cited and checked as the lead. The next kickoff brief shows a lesson once a steward has published it.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
