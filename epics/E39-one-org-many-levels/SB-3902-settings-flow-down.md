---
title: "SB-3902: Settings flow down"
summary: "Every team space has to repeat its department's glossary, start-here list and agents."
---

# SB-3902: Settings flow down

| Field | Value |
|---|---|
| Epic | [E39 One Org, Many Levels](README.md) |
| Wave / slice | W38 / S38.1 |
| Stories | US-39.1 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-24 |
| Depends on | SB-3901 |
| Status | Done |

## Why

Every team space has to repeat its department's glossary, start-here list and agents.

## Acceptance criteria

- [x] A child without its own `glossary`, `start_here`, `review_after_days` or `agents` takes its parent's; the child's own term wins
- [x] A child can only tighten `label` and `retention_days`
- [x] A parent's stewards are stewards of each child for escalations and roll-ups, not for publishing its notes

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Manifest
- [x] Spaces
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. A team takes its department's template, label and retention at check time, and its agents filtered by its template; `Tenant.Effective` merges glossaries (the team's term wins), joins start-here lists (the department's first, at most ten) and takes the review interval, used by the cell for the glossary, start here, review and verify paths. TestTreeFlowsDown, TestTreeThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
