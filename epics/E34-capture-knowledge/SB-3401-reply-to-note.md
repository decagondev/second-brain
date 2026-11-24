---
title: "SB-3401: Reply to note"
summary: "A steward's good reply goes back to one person and never reaches the index."
---

# SB-3401: Reply to note

| Field | Value |
|---|---|
| Epic | [E34 Capture Knowledge](README.md) |
| Wave / slice | W33 / S33.1 |
| Stories | US-34.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-07, LAW-11, LAW-16 |
| Depends on | SB-1705 |
| Status | Done |

## Why

A steward's good reply goes back to one person and never reaches the index.

## Acceptance criteria

- [x] Publish as a note on a steward's reply publishes it through the note flow, with its cites as sources
- [x] The asker is never named; the question's text goes only if the steward keeps it
- [x] Audited; the next ask in the space finds it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] Steward screen
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-24. The note API was already there (SB-1705); this adds **Keep the question in the note** (`keep_question`, off by default, kept on a republish) to the steward screen's **Publish as note**. The note is the title, the reply and its cites as sources, never the asker. Publishing runs through the tool bus, which audits it. Tests: `TestNoteKeepsQuestionAndComesDue`, the running cell's `TestAnswerOnceThroughARunningCell` (the next ask is answered from the note), web `capture.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
