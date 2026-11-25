---
title: "SB-5005: Did it help?"
summary: "Fixes, skills and agent changes ship, and nobody checks whether they moved anything."
---

# SB-5005: Did it help?

| Field | Value |
|---|---|
| Epic | [E50 Run AI Like a Business](README.md) |
| Wave / slice | W49 / S49.2 |
| Stories | US-50.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4701, SB-4507 |
| Status | Open |

## Why

Fixes, skills and agent changes ship, and nobody checks whether they moved anything.

## Acceptance criteria

- [ ] A shipped fix, skill, routing policy, policy or agent version schedules a check 4 weeks later comparing the 4 weeks before and after
- [ ] The numbers match the audit log and traces; fewer than 20 events either side says "not enough data"
- [ ] A check drafts a short result for the steward or owner and never undoes anything

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Check scheduling
- [ ] Before and after
- [ ] Result draft
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
