---
title: "SB-5105: Branch moderators"
summary: "A department steward can pause only the agents of their own space, not the teams below it."
---

# SB-5105: Branch moderators

| Field | Value |
|---|---|
| Epic | [E51 Guard What Goes In](README.md) |
| Wave / slice | W50 / S50.2 |
| Stories | US-51.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-kill-switches(7)`](../../manual/sb-kill-switches.7.md) |
| Laws | LAW-10, LAW-24 |
| Depends on | SB-4702, SB-3901 |
| Status | Done |

## Why

A department steward can pause only the agents of their own space, not the teams below it.

## Acceptance criteria

- [x] A steward of a space with children may pause any agent and review any shared skill, routine or rhythm in their branch
- [x] A child they can't open shows names and counts, never content
- [x] A sibling branch is refused, and every action is audited with the branch

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Branch scope for pause
- [x] Branch review list
- [x] Audit
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `Stewards.PauseVia` is the old `MayPause` (owner, or steward of the space) plus a walk up `Parent`: a steward of a space above may pause, and the space they moderate from is returned, so `Cell.pauseAgent` adds "(branch moderator of NAME)" to the audited reason. `GET /v1/spaces/NAME/branch` lists every space below, depth first, with its agents and whether each is paused there, its skill and rhythm names and `shared_routines` (`Stewards.SharedRoutineCount`): names and counts only. A sibling's steward is refused. While `branch-moderators` is in pilot (SB-5106) only the pilot group moderates. Tested in TestRollOutWithCareThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
