---
title: "SB-4106: A home for each space"
summary: "A space has no one place that shows what's happening in it."
---

# SB-4106: A home for each space

| Field | Value |
|---|---|
| Epic | [E41 Run the Company](README.md) |
| Wave / slice | W40 / S40.2 |
| Stories | US-41.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01 |
| Depends on | SB-3906 |
| Status | Done |

## Why

A space has no one place that shows what's happening in it.

## Acceptance criteria

- [x] The Start tab shows, in order, policies to read, goals this quarter, the latest update (and the teams'), open requests and what changed this week
- [x] Each part is one existing call checked as the reader, hidden when empty
- [x] No new page

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. The Answer tab with a space and no conversation is the space's home: Policies to read, Goals, Latest updates (the space's own then its teams'), Ask the department, What changed this week, then start here and who's who; each an existing call checked as the reader and hidden when empty; no new page. Tests: company.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
