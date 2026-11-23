---
title: "SB-2504: Counts on the tabs"
summary: "A steward can't see how much is waiting without opening each tab."
---

# SB-2504: Counts on the tabs

| Field | Value |
|---|---|
| Epic | [E25 Easier to Steward](README.md) |
| Wave / slice | W24 / S24.2 |
| Stories | US-25.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Done |

## Why

A steward can't see how much is waiting without opening each tab.

## Acceptance criteria

- [x] The Questions tab and the review section show how many wait, from the existing endpoints, read on load and every five minutes
- [x] Counts are said in words for screen readers
- [x] A count matches its list; axe

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S24.2; see manual/sb-ask.3.md

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
