---
title: "SB-3805: Daily brief"
summary: "The digest comes once a week, which is too slow for some people."
---

# SB-3805: Daily brief

| Field | Value |
|---|---|
| Epic | [E38 Work Where People Are](README.md) |
| Wave / slice | W37 / S37.2 |
| Stories | US-38.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07, LAW-12 |
| Depends on | SB-2103 |
| Status | Open |

## Why

The digest comes once a week, which is too slow for some people.

## Acceptance criteria

- [ ] Notification settings take `digest: daily`, which sends a short message each working morning at 08:00 instead of Monday's
- [ ] It holds what changed in their spaces and their watched questions
- [ ] A quiet day sends nothing; weekly people are unchanged

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Notify
- [ ] Settings
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
