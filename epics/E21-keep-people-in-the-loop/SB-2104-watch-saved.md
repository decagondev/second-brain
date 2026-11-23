---
title: "SB-2104: Watch a saved question"
summary: "A saved question doesn't say when its sources change."
---

# SB-2104: Watch a saved question

| Field | Value |
|---|---|
| Epic | [E21 Keep People in the Loop](README.md) |
| Wave / slice | W20 / S20.2 |
| Stories | US-21.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-2101 |
| Status | Done |

## Why

A saved question doesn't say when its sources change.

## Acceptance criteria

- [x] A saved question gains Watch; weekly the cell re-asks it as the person and compares the cited documents and versions with last week's
- [x] When they changed, the person gets "A source of your saved question changed" and a link; a changed cite they can no longer open counts but isn't named
- [x] No answer is stored, only the cite set's hash

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Watch flag on saved questions
- [x] Weekly re-ask
- [x] UI toggle
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: Done. `PUT /v1/saved/{id}/watch` and a Watch checkbox in the Saved tab. Watching runs the question as a search as its owner and keeps only `ask.SourcesHash` of the cites (each with its version), never a result or an answer; the weekly part searches again as the owner and tells them "The sources of N of your saved questions changed" with `/?tab=saved`, then keeps the new hash. A source they can't open now drops out of the set, so it counts and isn't named. Search, not the model, so a CPU box isn't loaded by watches. Covered by the digest law (one watch message, hash only, no source named) and `TestWatchASavedQuestionThroughARunningCell`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
