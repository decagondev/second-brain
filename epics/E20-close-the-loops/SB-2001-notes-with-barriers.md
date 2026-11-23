---
title: "SB-2001: Notes in cells with information barriers"
summary: "In a cell with information barriers, the bus places a call's target before it runs, and a note that doesn't exist yet can't be placed, so notes are refused."
---

# SB-2001: Notes in cells with information barriers

| Field | Value |
|---|---|
| Epic | [E20 Close the Loops](README.md) |
| Wave / slice | W19 / S19.1 |
| Stories | US-20.1 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | LAW-04, LAW-22 |
| Depends on | — |
| Status | Done |

## Why

In a cell with information barriers, the bus places a call's target before it runs, and a note that doesn't exist yet can't be placed, so notes are refused.

## Acceptance criteria

- [x] A `note-publish` target is placed by the space it is published in, which is known before the write
- [x] A steward on one side of a barrier can publish a note in their space; the barrier still refuses a target in a space across it
- [x] The W16 refusal becomes an allow-for-the-space and deny-across-the-barrier test

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Placement for notes
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built 2026-11-23. `barrier.Cell.Target` places a note target (`notes/SPACE/qs_ID.md`) by its path before it is written: the spaces whose scope holds the path, with the steward as owner; any other target is still located. A steward publishes in their own space; a note in a space across a barrier is refused and nothing is written; an unreadable registry refuses (`TestLAW22_NotesPublishOnTheirSideOfABarrier` with the real bus, Cedar and folder connector; self-tests catch a cell that refuses every note, places it nowhere, or ignores an unreadable registry). Manual: sb-spaces(7), sb-connector(3), sb-agents(7).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
