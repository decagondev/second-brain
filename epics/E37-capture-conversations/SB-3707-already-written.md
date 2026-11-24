---
title: "SB-3707: Already written?"
summary: "A steward can publish a note that repeats a document already in the space."
---

# SB-3707: Already written?

| Field | Value |
|---|---|
| Epic | [E37 Capture Conversations](README.md) |
| Wave / slice | W36 / S36.2 |
| Stories | US-37.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | SB-3402 |
| Status | Done |

## Why

A steward can publish a note that repeats a document already in the space.

## Acceptance criteria

- [x] Before Publish as a note, the steward sees up to 3 documents in the space that read the same
- [x] Each is checked as the steward; one they can't open isn't named
- [x] It's a warning only

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Stewards
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

The note preview names up to 3 documents of the space that read the same, checked as the steward

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
