---
title: "SB-2803: Never cited"
summary: "Nobody sees which documents answers never use."
---

# SB-2803: Never cited

| Field | Value |
|---|---|
| Epic | [E28 A Tidy Corpus](README.md) |
| Wave / slice | W27 / S27.1 |
| Stories | US-28.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | — |
| Status | Done |

## Why

Nobody sees which documents answers never use.

## Acceptance criteria

- [x] `GET /v1/spaces/NAME/review?section=uncited` lists documents no answer cited in 180 days and older than 90 days, at most 100, oldest first
- [x] Titles only for documents the steward can open; never question text
- [x] A document cited once in the window isn't listed; a hidden one is a count only

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S27.1; tested in internal/core/ask/tidy_test.go, tests/cell/retire_test.go and web/src/tidy.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
