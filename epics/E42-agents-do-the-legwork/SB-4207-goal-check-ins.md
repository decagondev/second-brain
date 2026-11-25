---
title: "SB-4207: Goal check-ins"
summary: "Goals go quiet: their status is set once a quarter and nobody notices when the work behind them stops."
---

# SB-4207: Goal check-ins

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.2 |
| Stories | US-42.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-16, LAW-24 |
| Depends on | SB-4101 |
| Status | Done |

## Why

Goals go quiet: their status is set once a quarter and nobody notices when the work behind them stops.

## Acceptance criteria

- [x] Each Monday, a goal gets a drafted check-in for its owner from the tickets, decisions, updates and documents that cite it, with a suggested status
- [x] The owner publishes it as a note under the goal or discards it; nothing is published without them
- [x] A goal with no published check-in for 3 weeks is flagged on its owner's inbox and its space's home; only counts roll up

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent and prompt
- [x] Notify
- [x] Notes
- [x] UI
- [x] Eval suite
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. Each Monday Cell.runCheckins drafts a check-in (DraftCheckin, Thread.Goal) as the goal's owner when they are one active user and a steward; a published check-in goes under notes/NAME/goals/. ask.LastCheckin/Silent; GET /v1/spaces/NAME/goals marks silent goals; the owner gets a checkin notification. The owner's inbox comes with SB-4301. Tests: TestSilentGoals, TestFollowUpThroughARunningCell, agent-goal-check-in suite.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
