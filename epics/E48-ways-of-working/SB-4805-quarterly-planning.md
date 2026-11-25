---
title: "SB-4805: Quarterly planning"
summary: "Next quarter's goals are drafted from memory, and overlaps between departments surface late."
---

# SB-4805: Quarterly planning

| Field | Value |
|---|---|
| Epic | [E48 Ways of Working](README.md) |
| Wave / slice | W47 / S47.2 |
| Stories | US-48.2 |
| Manual | [`sb-departments(7)`](../../manual/sb-departments.7.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-16, LAW-24 |
| Depends on | SB-4801, SB-4207, SB-3905 |
| Status | Done |

## Why

Next quarter's goals are drafted from memory, and overlaps between departments surface late.

## Acceptance criteria

- [x] Two weeks before quarter end each space gets cited draft goals from check-ins, at-risk goals and request themes
- [x] A parent's stewards see children's proposed goals rolled up; a child they can't open shows counts only
- [x] An overlap between departments is a flag, never a merge; goals start only when each steward publishes

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Planning rhythm
- [x] Goal draft agent and eval suite
- [x] Roll-up and overlaps
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. The built-in `quarterly-planning` rhythm's day-0 step uses the `goal-plan` drafter (`internal/core/ask/planning.go`); `Rhythms.Tick` starts it in every space 14 days before a quarter ends (`quarterly`), drafted as the space's first active steward. The draft carries over this quarter's unfinished goals of the space (at most 10, at risk or off track flagged), each cited, and counts the quarter's requests by category; the proposed goals are kept on the run. `GET /v1/spaces/NAME/planning` (`Rhythms.Planning`) rolls up the latest draft of the space and its teams for its stewards: a goal shown only when they can open its cite, the rest counted in `hidden`, overlaps flagged by a shared glossary term or two shared words. The draft is put together by code from goals the steward can open, not by a model, so it has no agent definition or eval suite of its own (as S47.1's kickoff brief). Request themes are counts by category; voice-of-the-customer themes are not drawn in. Tested in TestPlanningOneOnOneAndTreePlanThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
