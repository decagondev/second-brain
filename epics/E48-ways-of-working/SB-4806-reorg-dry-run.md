---
title: "SB-4806: Reorg dry run"
summary: "A move in the tree changes inheritance, escalation and agents, and nobody sees it before it happens."
---

# SB-4806: Reorg dry run

| Field | Value |
|---|---|
| Epic | [E48 Ways of Working](README.md) |
| Wave / slice | W47 / S47.2 |
| Stories | US-48.2 |
| Manual | [`sb-space(1)`](../../manual/sb-space.1.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-11, LAW-24 |
| Depends on | SB-3902, SB-3907 |
| Status | Done |

## Why

A move in the tree changes inheritance, escalation and agents, and nobody sees it before it happens.

## Acceptance criteria

- [x] `sb tree move --plan` and `sb tree merge --plan` list inherited settings, escalation paths, roll-ups, agents, routines, waits, rhythms and lost ownership
- [x] The plan matches the tree after the manifest diff is applied in the test cell; nothing changes on `--plan`
- [x] A move that would widen any access is refused with the path

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Tree diff
- [x] CLI
- [x] Manifest diff proposal
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `internal/core/reorg/tree.go`: `Move` and `Merge` make the manifest the change would leave, check it with every rule of `sb tenant check`, and list for the space, its teams and its old and new parent: the parent, escalation path and roll-up before and after, and each inherited setting gained or lost (glossary, skills, rhythms, start-here, agents, label, retention). A label that loosens, a retention that grows or a merge whose members differ is `ErrWiden` with the path (LAW-24). `sb tree move|merge FILE … --plan [--write OUT]` runs it offline; `GET /v1/admin/tree/plan` runs it against the applied manifest and adds `in_flight` per space (running rhythms, open and waiting queue items, open projects, routines, triggered agents). A move or merge changes no space's members, so nobody loses something they own; a merge needs the same members. The plan's manifest goes through the usual review; nothing applies. Tested in TestMovePlansWhatChanges, TestMergeFoldsOneSpaceIntoAnother, TestTreeRefusesWhatWouldWiden, TestTreePlan and the cell test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
