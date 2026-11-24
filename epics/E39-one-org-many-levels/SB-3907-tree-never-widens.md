---
title: "SB-3907: The tree never widens"
summary: "Nothing yet proves that a place in the tree never widens what anyone sees."
---

# SB-3907: The tree never widens

| Field | Value |
|---|---|
| Epic | [E39 One Org, Many Levels](README.md) |
| Wave / slice | W38 / S38.2 |
| Stories | US-39.2 |
| Manual | [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-24 |
| Depends on | SB-3903, SB-3905 |
| Status | Done |

## Why

Nothing yet proves that a place in the tree never widens what anyone sees.

## Acceptance criteria

- [x] LAW-24 is in sb-laws(7): a space's place in the tree never lets anyone see, count or be told about a document they couldn't see from the space that holds it
- [x] Its law test runs 100 generated trees through branch asks, roll-ups, inherited start-here lists and escalations
- [x] The selftest unions members up the tree and must go red

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Manual
- [x] Law test
- [x] Selftest
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. LAW-24 in sb-laws(7). `laws/law24.go`: `TestLAW24_TheTreeNeverWidens` runs 300 generated trees through branch asks (`spaces.Teams`, the Issuer, `retrieve.ContextUpTo`) against each space's own ask, and the manifest's settings that flow down; `TestSelf_LAW24CatchesAWideTree` opens every team to the branch and goes red.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
