---
title: "SB-2605: Doctor says how to fix it"
summary: "The doctor says what is wrong but not how to fix it."
---

# SB-2605: Doctor says how to fix it

| Field | Value |
|---|---|
| Epic | [E26 Easier to Operate](README.md) |
| Wave / slice | W25 / S25.2 |
| Stories | US-26.2 |
| Manual | [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | — |
| Depends on | — |
| Status | Open |

## Why

The doctor says what is wrong but not how to fix it.

## Acceptance criteria

- [ ] Each verdict that isn't OK gains a `fix`: one command or a runbook section, printed by `sb selfhost doctor` and shown in the console's Box section
- [ ] A test fails on a non-OK verdict with no fix, or a fix naming a missing runbook section

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Fixes
- [ ] Console
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
