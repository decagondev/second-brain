---
title: "SB-5004: Agent goals"
summary: "We measure each agent, but nobody says what good looks like for it."
---

# SB-5004: Agent goals

| Field | Value |
|---|---|
| Epic | [E50 Run AI Like a Business](README.md) |
| Wave / slice | W49 / S49.1 |
| Stories | US-50.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4701, SB-4707 |
| Status | Done |

## Why

We measure each agent, but nobody says what good looks like for it.

## Acceptance criteria

- [x] An owner sets a one-sentence `job:` and up to 3 targets on value numbers only
- [x] The registry shows each target against the last 4 weeks, and a miss two weeks running tells the owner once
- [x] The agent's record lists every target and its history; meeting a target changes nothing else

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Job and targets
- [x] Registry
- [x] Record
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: an agent's map in the manifest takes `job` (one sentence, at most 200 characters) and up to 3 `targets` on `published_as_is`, `discard_rate`, `hours_to_publish`, `helpful_rate` and `cost_per_published_usd`. `internal/core/ask/agentgoals.go`: `GoalViews` counts each target over the last 4 complete weeks for the value report's `goals`, `MissedTwice` tells the owner once per run of two missed weeks (kind `agent`), and the agent's record lists every target with 12 weeks. A target changes nothing the agent does. Tested in TestGoalViewsAndMissedTwice, TestLimitsAndAgentGoals and TestSpendWithPurposeThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
