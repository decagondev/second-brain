---
title: "SB-2702: Reviewed shows on the cite"
summary: "A steward's review mark doesn't reach the asker, so a reviewed document still looks stale."
---

# SB-2702: Reviewed shows on the cite

| Field | Value |
|---|---|
| Epic | [E27 Fresh Answers](README.md) |
| Wave / slice | W26 / S26.1 |
| Stories | US-27.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-13 |
| Depends on | SB-2501 |
| Status | Done |

## Why

A steward's review mark doesn't reach the asker, so a reviewed document still looks stale.

## Acceptance criteria

- [x] A cite with a review mark at its version carries `reviewed` and isn't stale until `review_after_days` passes from the mark
- [x] The chip says Reviewed DATE in words; the steward isn't named
- [x] Mark, then the cite isn't stale; a new version is stale again by its own age

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Cite field
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S26.1; tests green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
