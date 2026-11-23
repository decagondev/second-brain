---
title: "SB-2604: Check the config first"
summary: "A typo in the cell's config shows up only when a service fails to start."
---

# SB-2604: Check the config first

| Field | Value |
|---|---|
| Epic | [E26 Easier to Operate](README.md) |
| Wave / slice | W25 / S25.1 |
| Stories | US-26.1 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb(1)`](../../manual/sb.1.md), [`sb-selfhost(7)`](../../manual/sb-selfhost.7.md) |
| Laws | — |
| Depends on | — |
| Status | Done |

## Why

A typo in the cell's config shows up only when a service fails to start.

## Acceptance criteria

- [x] `sb cell check-config FILE` runs the services' own check and confirms each vault reference exists with owner-only permissions, without reading or printing it
- [x] `update.sh` runs it before any restart and stops with the line at fault
- [x] A typo, a missing secret and a world-readable secret each stop the update before a restart

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] CLI
- [x] update.sh
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S25.1: sb cell check-config, run by update.sh before any restart; vault files written owner-only

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
