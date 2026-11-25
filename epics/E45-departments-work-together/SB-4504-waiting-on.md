---
title: "SB-4504: Waiting on another department"
summary: "Work blocked on another department is invisible to both, and nobody above them hears until it's late."
---

# SB-4504: Waiting on another department

| Field | Value |
|---|---|
| Epic | [E45 Departments Work Together](README.md) |
| Wave / slice | W44 / S44.2 |
| Stories | US-45.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-24 |
| Depends on | SB-4104, SB-4101, SB-4203 |
| Status | Done |

## Why

Work blocked on another department is invisible to both, and nobody above them hears until it's late.

## Acceptance criteria

- [x] A request, goal or action item can be marked waiting on another space with a reason and a date
- [x] Both homes list it, checked as the reader, and roll-ups count cross-space waits
- [x] A wait a working day past its date goes once to the nearest common parent's stewards

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Field and API
- [x] Job
- [x] UI
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `ask.Wait` (`on`, `reason`, `until`, `by`, `at`) on `StewardQuestion.Waiting` (requests) and `Action.Waiting`; a goal waits through `waiting_on`, `waiting_reason`, `waiting_until` in its front matter (`GoalMeta.Waiting`, `goalWait`). `PUT|DELETE|GET /v1/spaces/NAME/waits` (`Stewards.SetWait`, `ClearWait`, `Waits`): the sender or a steward marks, either side clears, and each item is named only to someone who can see it (a request's category and the reason also to a steward of the space it waits on), the rest counted. `TeamRollup.Waiting` counts a team's waits. The notify job (`Job.Waits`, `Stewards.OverdueWaits`) tells the stewards of the two spaces' nearest common parent (`commonParent`), or both with none, once per wait and date (clock `wait-KIND-ID-DATE`). Home card **Waiting** and **Waiting on…** on a request in the steward queue (`web/src/Across.tsx`). TestWorkAcrossDepartmentsThroughARunningCell, TestWaitOverdueAndCommonParent, TestOverdueWaitsGoUpOnce, across.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
