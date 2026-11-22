---
title: "SB-1709: Documents to review for stewards"
summary: "The most damaging answer is a confident one from an old document, and nobody sees which old documents answer the most questions."
---

# SB-1709: Documents to review for stewards

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.4 |
| Stories | US-17.4 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-14, LAW-16 |
| Depends on | — |
| Status | Done |

## Why

The most damaging answer is a confident one from an old document, and nobody sees which old documents answer the most questions.

## Acceptance criteria

- [x] `review_after_days` in sb-space(5), default 365
- [x] The steward's screen lists documents held by the space cited at least five times in 30 days and older than the threshold, plus any marked as a wrong citation, from the traces
- [x] Each item is checked live as the steward; one they can't open is counted, with its owner if any, never named
- [x] Cites gain `age_days`, and the UI shows a document's age on its chip past the threshold

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Counts from traces
- [x] Steward section and chip age
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S16.4): `review_after_days` in sb-space(5) (30 to 3650, default 365, stored as 0 so the manifest reads back the same). `GET /v1/spaces/NAME/review`, for the space's stewards by the existing `review_space` action, counts each document's cites in the last 30 days of traces and its `wrong_citation` marks in feedback, keeps the documents the space holds that were cited at least 5 times and are older than the threshold, or were marked, and checks each as the steward: shown with its counts, or only an entry in `hidden` with its owner. Every cite gains `age_days` and `stale` (past the shortest threshold of the spaces holding it); the UI shows "N days old" on a stale chip, in its accessible name, and the steward screen lists Documents to review. Tests: `TestLAW01_TheReviewListNamesOnlyWhatTheStewardCanOpen`, UI tests with axe, and the running cell refuses the list to a non-steward.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
