---
title: "SB-3604: Unhelpful clears the mark"
summary: "A reviewed document stays off the review list even when people find its answers unhelpful."
---

# SB-3604: Unhelpful clears the mark

| Field | Value |
|---|---|
| Epic | [E36 Trust Each Answer](README.md) |
| Wave / slice | W35 / S35.1 |
| Stories | US-36.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07 |
| Depends on | SB-2501 |
| Status | Done |

## Why

A reviewed document stays off the review list even when people find its answers unhelpful.

## Acceptance criteria

- [x] 3 unhelpful ratings of answers citing a marked document since its mark put it back on the review list, with why
- [x] Ratings before the mark don't count
- [x] No asker is named

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Review list
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Three unhelpful ratings since a mark list the document again

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
