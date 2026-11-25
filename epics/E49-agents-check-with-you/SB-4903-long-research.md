---
title: "SB-4903: Long research with checkpoints"
summary: "Hard questions need more than one pass, but a long run nobody can steer wastes money."
---

# SB-4903: Long research with checkpoints

| Field | Value |
|---|---|
| Epic | [E49 Agents That Check With You](README.md) |
| Wave / slice | W48 / S48.1 |
| Stories | US-49.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | LAW-01, LAW-26 |
| Depends on | SB-4201, SB-4105 |
| Status | Open |

## Why

Hard questions need more than one pass, but a long run nobody can steer wastes money.

## Acceptance criteria

- [ ] `long: true` shows the plan and cost estimate first, and nothing runs before it is approved
- [ ] At most 6 stages and 4 hours; at each checkpoint the asker can steer, stop or let it go on
- [ ] A run stops at `research.max_usd` with what it has, and a document the asker lost mid-run is dropped at the next stage

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Plan approval
- [ ] Stages and checkpoints
- [ ] Cost ceiling
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
