---
title: "SB-3307: Suggested questions"
summary: "A new person facing an empty ask box doesn't know what to ask in a space."
---

# SB-3307: Suggested questions

| Field | Value |
|---|---|
| Epic | [E33 Ask Better](README.md) |
| Wave / slice | W32 / S32.2 |
| Stories | US-33.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-11, LAW-19 |
| Depends on | SB-2503 |
| Status | Done |

## Why

A new person facing an empty ask box doesn't know what to ask in a space.

## Acceptance criteria

- [x] A space's stewards keep up to 8 starter questions of at most 120 characters, one record per space, audited
- [x] An empty ask box in that space shows them as buttons for its members; a company-wide ask shows none
- [x] A ninth question or a non-steward gets refused

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Store and API
- [x] Steward screen and UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

ask.Suggested and SuggestedStore (cell_docs kind suggested, one per space; no new store), Stewards.SetSuggested (stewards only, at most 8 of 120 characters, audited with the list's SHA-256) and SuggestedFor (members only, else EPERM); GET and PUT /v1/spaces/NAME/suggested; the ask box shows them as buttons in a space when empty, and the steward screen edits them. Tests: tests/cell TestSuggestedQuestionsThroughARunningCell, web knowwhere.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
