---
title: "SB-3105: Coming due"
summary: "The review list shows what is overdue, not what falls due soon."
---

# SB-3105: Coming due

| Field | Value |
|---|---|
| Epic | [E31 Fill the Gaps](README.md) |
| Wave / slice | W30 / S30.2 |
| Stories | US-31.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Done |

## Why

The review list shows what is overdue, not what falls due soon.

## Acceptance criteria

- [x] `review?section=due` lists cited documents that turn stale in the next 14 days, soonest first, by review_after_days or their own date
- [x] Reviewed, still right works on them
- [x] Due in 10 days is listed; 20 days isn't

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Review section
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

review?section=due lists relied-on documents that turn stale in the next 14 days, soonest first, with due_in_days; Reviewed, still right works on them and a mark moves the date on, except a document's own review-by date

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
