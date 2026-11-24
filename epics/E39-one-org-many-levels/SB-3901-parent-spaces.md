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
| Status | Open |

## Why

Spaces are flat, so a team is just another space beside its department.

## Acceptance criteria

- [ ] sb-space(5) gains `parent: NAME`; the tree is at most three levels with no cycles
- [ ] `sb tenant check` refuses a deeper tree, a missing parent, a cycle or a looser child, with the line
- [ ] Members aren't inherited; the registry knows each space's parent and depth

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Manifest and check
- [ ] Registry
- [ ] Reorg preview
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
