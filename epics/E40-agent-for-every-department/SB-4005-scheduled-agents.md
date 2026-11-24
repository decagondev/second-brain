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
| Status | Done |

## Why

Every agent runs only when someone remembers to ask.

## Acceptance criteria

- [x] A space can set `brief: { agent: NAME, weekly: monday }`, one a space
- [x] The notify job runs it weekly as the first active named steward, with a fixed question; the answer waits in the steward queue as a draft
- [x] A space with no active steward, or over budget, runs nothing and its stewards are told

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Manifest
- [x] Notify job
- [x] Steward queue
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

brief: { agent, weekly } runs weekly as the first active steward; the answer waits as a draft only that steward can read; budget, kill or no steward runs nothing and tells the stewards

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
