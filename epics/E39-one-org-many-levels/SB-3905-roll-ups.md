---
title: "SB-3905: Roll-ups for leaders"
summary: "A department's steward can't see how its teams are doing."
---

# SB-3905: Roll-ups for leaders

| Field | Value |
|---|---|
| Epic | [E39 One Org, Many Levels](README.md) |
| Wave / slice | W38 / S38.2 |
| Stories | US-39.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07, LAW-24 |
| Depends on | SB-3506 |
| Status | Done |

## Why

A department's steward can't see how its teams are doing.

## Acceptance criteria

- [x] A department's review list gains Teams: each child's counts of to review, overdue, open questions and found-nothing
- [x] Usage by space nests teams under their department; rows under 5 asks show fewer than 5
- [x] Titles and questions never roll up

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Review list
- [x] Report
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `ReviewResponse.Teams`, `TeamRollup`, `Stewards.teams` (`Children`, `ReviewAfter`): per team, the documents to review its department's steward can open, open and overdue questions, and this week's asks and found-nothing, or fewer than 5. `ask.Nest`, `report.TreeSource`, `usage.ParentOf`: the weekly report shows `finance / ap`. Web `TeamsTable`. TestTreeThroughARunningCell, tree.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
