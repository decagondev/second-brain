---
title: "SB-4005: Agents on a schedule"
summary: "Every agent runs only when someone remembers to ask."
---

# SB-4005: Agents on a schedule

| Field | Value |
|---|---|
| Epic | [E40 An Agent for Every Department](README.md) |
| Wave / slice | W39 / S39.2 |
| Stories | US-40.2 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-07, LAW-12, LAW-16 |
| Depends on | SB-3703 |
| Status | Open |

## Why

Every agent runs only when someone remembers to ask.

## Acceptance criteria

- [ ] A space can set `brief: { agent: NAME, weekly: monday }`, one a space
- [ ] The notify job runs it weekly as the first active named steward, with a fixed question; the answer waits in the steward queue as a draft
- [ ] A space with no active steward, or over budget, runs nothing and its stewards are told

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Manifest
- [ ] Notify job
- [ ] Steward queue
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
