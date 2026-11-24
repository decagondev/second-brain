---
title: "SB-3901: Parent spaces"
summary: "Spaces are flat, so a team is just another space beside its department."
---

# SB-3901: Parent spaces

| Field | Value |
|---|---|
| Epic | [E39 One Org, Many Levels](README.md) |
| Wave / slice | W38 / S38.1 |
| Stories | US-39.1 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-21, LAW-24 |
| Depends on | SB-1206 |
| Status | Done |

## Why

Spaces are flat, so a team is just another space beside its department.

## Acceptance criteria

- [x] sb-space(5) gains `parent: NAME`; the tree is at most three levels with no cycles
- [x] `sb tenant check` refuses a deeper tree, a missing parent, a cycle or a looser child, with the line
- [x] Members aren't inherited; the registry knows each space's parent and depth

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Manifest and check
- [x] Registry
- [x] Reorg preview
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. sb-space(5) `parent`; `manifest.Space.Parent`, `checker.treeChecks` (rule 17: a missing parent, the space itself, a fourth level or a cycle), a team at least as strict as its department (`labelRank`), `Tenant.Depth`, `Children`, `Branch`, `Up`; `ports.Space.Parent` in the registry. Members are never inherited. TestTreeFlowsDown, TestTreeRefused, TestTreeStricterChildIsFine.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
