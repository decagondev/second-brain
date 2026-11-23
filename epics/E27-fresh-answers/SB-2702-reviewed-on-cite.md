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
| Status | Open |

## Why

A steward's review mark doesn't reach the asker, so a reviewed document still looks stale.

## Acceptance criteria

- [ ] A cite with a review mark at its version carries `reviewed` and isn't stale until `review_after_days` passes from the mark
- [ ] The chip says Reviewed DATE in words; the steward isn't named
- [ ] Mark, then the cite isn't stale; a new version is stale again by its own age

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Cite field
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
