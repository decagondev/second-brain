---
title: "SB-3205: Answer time watch"
summary: "Nobody watches how long answers take."
---

# SB-3205: Answer time watch

| Field | Value |
|---|---|
| Epic | [E32 Run Without Surprises](README.md) |
| Wave / slice | W31 / S31.2 |
| Stories | US-32.2 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07 |
| Depends on | SB-2906 |
| Status | Open |

## Why

Nobody watches how long answers take.

## Acceptance criteria

- [ ] The usage ledger records each model call's duration; health and the weekly report show this week's and last week's 95th percentile of seconds
- [ ] It warns at more than 1.5 times last week's
- [ ] A doubled week warns; a flat one doesn't

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Ledger
- [ ] Health and report
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
