---
title: "SB-4302: My routines"
summary: "People ask the same question every Monday by hand."
---

# SB-4302: My routines

| Field | Value |
|---|---|
| Epic | [E43 Work That Comes to You](README.md) |
| Wave / slice | W42 / S42.1 |
| Stories | US-43.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-01, LAW-12, LAW-26 |
| Depends on | SB-3805 |
| Status | Done |

## Why

People ask the same question every Monday by hand.

## Acceptance criteria

- [x] A saved question can become a routine, daily on working mornings or weekly, at most 5 per person
- [x] The notify job asks it as the person and sends the cited answer, what changed first, to their chat or email
- [x] A routine pauses at the person's space budget and stops when they are deprovisioned

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Records
- [x] Notify
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. SavedQuestion.Routine, ask.Questions.SetRoutine (MaxRoutines 5), PUT /v1/saved/{id}/routine; ask.Legwork.RunRoutines asks each due routine (RoutineSlot, RoutineDue in the digest zone) as its owner now and keeps a routine result with Changed and the new sources' claims first; a spent budget pauses it, an inactive owner is never asked for. notify.KindRoutine, Job.Routines. Stores: Routines in the memory and Postgres trace stores. Web: Routine select on the Saved tab. Tests: TestWorkComesToYouThroughARunningCell, inbox.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
