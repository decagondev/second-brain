---
title: "SB-1706: Start here: a reading list per space"
summary: "New joiners lose weeks to 'where do I start', and every team's reading list lives in someone's head."
---

# SB-1706: Start here: a reading list per space

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.3 |
| Stories | US-17.3 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-tenant(1)`](../../manual/sb-tenant.1.md) |
| Laws | LAW-01, LAW-03, LAW-16 |
| Depends on | — |
| Status | Proposed |

## Why

New joiners lose weeks to "where do I start", and every team's reading list lives in someone's head.

## Acceptance criteria

- [ ] `start_here` in sb-space(5): up to ten cites or folder paths, in order; a change is a steward's proposal like any space change
- [ ] Members see Start here in the space; each item is checked live, and one they can't open, or whose check is Unknown, is counted, not named
- [ ] `sb tenant verify` reports any item no member of the space can open

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Manifest field and check
- [ ] UI section
- [ ] Verify check and tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Proposed 2026-11-22 with W16 ([09-team-knowledge.md](../../docs/09-team-knowledge.md)). Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
