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
| Status | Done |

## Why

New joiners lose weeks to "where do I start", and every team's reading list lives in someone's head.

## Acceptance criteria

- [x] `start_here` in sb-space(5): up to ten cites or folder paths, in order; a change is a steward's proposal like any space change
- [x] Members see Start here in the space; each item is checked live, and one they can't open, or whose check is Unknown, is counted, not named
- [x] `sb tenant verify` reports any item no member of the space can open

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Manifest field and check
- [x] UI section
- [x] Verify check and tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S16.3): `start_here` in sb-space(5): up to ten items, each a document `KIND:ID` (newest version) or a folder `KIND:/Folder/`, each kind in the space's scope; checked, formatted, in the schema and the plan's space attributes, and changed like any space field through a steward's proposal. Chunks now carry their `Location`, and both indexes implement `ports.DocLookup` (by source, by location under a folder, by owner). `GET /v1/start?space=NAME` resolves the list for a member with `retrieve.Allowed`, so a document they can't open, or whose check is Unknown or late, or that isn't indexed, is counted in `hidden`, never named; a folder lists its documents by location, ten items at most. The web UI shows Start here when a space is chosen. `sb tenant verify` gains a `START` check (`verify.StartTarget`): an item no active member of the space can open fails, named by space and position. Tests: `TestLAW01_StartHereShowsOnlyWhatAMemberCanOpen` (its self-test catches Unknown as Allow), `TestStartHere` in core and in verify, and a UI test with axe.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
