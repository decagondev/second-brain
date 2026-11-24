---
title: "SB-3403: Notes come due"
summary: "A published note never comes up for review."
---

# SB-3403: Notes come due

| Field | Value |
|---|---|
| Epic | [E34 Capture Knowledge](README.md) |
| Wave / slice | W33 / S33.1 |
| Stories | US-34.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-13 |
| Depends on | SB-2705 |
| Status | Done |

## Why

A published note never comes up for review.

## Acceptance criteria

- [x] A published note gets review_by 180 days ahead
- [x] It shows in Documents to review and Coming due like any document
- [x] Republishing resets the date

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Note flow
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-24. A note's front matter carries `review_by:` 180 days from publishing, which ingest already reads (SB-2705), so it joins Documents to review and Coming due. A republish writes the date again: the folder connector rewrites a note only when everything after its front matter is unchanged, and never touches its ACL file there. Tests: `TestNoteRedates`, `TestNoteKeepsQuestionAndComesDue`, the running cell's note check.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
