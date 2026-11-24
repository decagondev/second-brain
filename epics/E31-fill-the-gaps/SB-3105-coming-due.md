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
| Status | Open |

## Why

The review list shows what is overdue, not what falls due soon.

## Acceptance criteria

- [ ] `review?section=due` lists cited documents that turn stale in the next 14 days, soonest first, by review_after_days or their own date
- [ ] Reviewed, still right works on them
- [ ] Due in 10 days is listed; 20 days isn't

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Review section
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
