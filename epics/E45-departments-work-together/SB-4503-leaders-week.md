---
title: "SB-4503: The leader's week"
summary: "A leader assembles the week from each team's updates, goals and queues by hand."
---

# SB-4503: The leader's week

| Field | Value |
|---|---|
| Epic | [E45 Departments Work Together](README.md) |
| Wave / slice | W44 / S44.1 |
| Stories | US-45.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-16, LAW-24 |
| Depends on | SB-4102, SB-4207, SB-3905 |
| Status | Done |

## Why

A leader assembles the week from each team's updates, goals and queues by hand.

## Acceptance criteria

- [x] Each Monday the stewards of a space with children get a drafted review of the branch
- [x] Every line is cited, or counted for a child the steward can't open
- [x] The steward publishes or discards it; `leader_week: false` turns it off

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent and prompt
- [x] Job
- [x] UI
- [x] Eval suite
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. sb-space(5) `leader_week` (default on; `manifest.Tenant.LeaderWeek`: a space with teams and stewards). The `leader-week` agent (prompt `leader-week`, base.cedar, agents.json, suite `agent-leader-week`) runs only there. Each Monday the notify job (`Job.Leaders`, `Job.RunLeader`) runs it as the space's first active steward as a branch ask over the space and the teams that steward is in (`Cell.runLeader`, `Cell.branchOf`), and its draft (kind `leader-week`) waits in the steward queue: the cited claims, then `By team (counts only)` with each team's open and overdue requests, open questions and budget share (`Stewards.teamCounts`, `Stewards.BudgetShare`). A space over its budget or without an active steward is told why, as briefs are. Published, it goes under `notes/NAME/updates/` as the space's update. TestLeaderWeekRunsOnMonday, TestLeaderWeekNeedsTeamsAndStewards and the cell test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
