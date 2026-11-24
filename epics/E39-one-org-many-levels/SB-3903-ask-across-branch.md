---
title: "SB-3903: Ask across a branch"
summary: "A department member can't ask across the teams they belong to in one go."
---

# SB-3903: Ask across a branch

| Field | Value |
|---|---|
| Epic | [E39 One Org, Many Levels](README.md) |
| Wave / slice | W38 / S38.1 |
| Stories | US-39.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-24 |
| Depends on | SB-3901 |
| Status | Done |

## Why

A department member can't ask across the teams they belong to in one go.

## Acceptance criteria

- [x] `?space=NAME&branch=true` runs the space checks once per space in the branch the asker is a member of, and unions what each allowed
- [x] A team the asker isn't in adds nothing and isn't named
- [x] Cites say which space each came from; without `branch` nothing changes

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Retrieval
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `?space=NAME&branch=true`: the auth middleware issues the asker's scope in each team under the space they are a member of (`spaces.Teams`, `tenancy.WithBranch`), and `retrieve.branchContext` runs retrieval and every check once per space and joins what each allowed. `GET /v1/spaces` gains `parent` and `teams`; the web switcher offers **NAME and its teams**. TestTreeThroughARunningCell, tree.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
