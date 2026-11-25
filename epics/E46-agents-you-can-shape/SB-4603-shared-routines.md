---
title: "SB-4603: Shared routines"
summary: "Every member of a team sets up the same routine by hand, or doesn't."
---

# SB-4603: Shared routines

| Field | Value |
|---|---|
| Epic | [E46 Agents You Can Shape](README.md) |
| Wave / slice | W45 / S45.1 |
| Stories | US-46.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-12, LAW-26 |
| Depends on | SB-4302 |
| Status | Done |

## Why

Every member of a team sets up the same routine by hand, or doesn't.

## Acceptance criteria

- [x] A steward shares a routine with the space and members opt in
- [x] Each run is a separate ask as each member, sent to them alone; the steward sees only how many opted in
- [x] It counts toward each member's 5 routines and stops for a member who leaves

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Share
- [x] Job
- [x] UI
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `SavedQuestion.Shared` (the space a steward shared their routine with) and `From` (a member's copy). `PUT|DELETE /v1/spaces/NAME/routines/{id}` (`Stewards.ShareRoutine`) for a steward's own routine in that space; `GET /v1/spaces/NAME/routines` (`SharedRoutines`) for members, with `opted_in` (a count, never who) only on the caller's own; `POST|DELETE …/join` (`JoinRoutine`) makes or removes the member's own copy, counted toward their 100 saved questions and 5 routines. A copy runs as any routine of theirs, as them; `RunRoutines` stops a copy whose routine is no longer shared, and `runRoutine` asks nothing for anyone no longer a member of the routine's space (`leftSpace`), stopping a copy. Every change is audited (`shared`, `unshared`, `joined`, `left`). No new store: the routines list every store already has. Web: **Shared routines** on the Start tab, **Share with the space** in the Saved tab. Tests: `tests/cell/shape_test.go` (a member can't share, the steward sees the count and none of the member's answers, the copy runs as the member and stops when they leave), `web/src/shape.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
