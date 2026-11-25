---
title: "SB-4304: Collections"
summary: "The documents for a piece of work are scattered, and each ask has to find them again."
---

# SB-4304: Collections

| Field | Value |
|---|---|
| Epic | [E43 Work That Comes to You](README.md) |
| Wave / slice | W42 / S42.2 |
| Stories | US-43.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-13 |
| Depends on | SB-3004 |
| Status | Done |

## Why

The documents for a piece of work are scattered, and each ask has to find them again.

## Acceptance criteria

- [x] A collection names up to 50 documents and saved threads, shared with named people, stored as ids only
- [x] **Ask this collection** scopes an ask to its documents; it cites nothing outside them
- [x] Each reader sees only the items they can open; others are counted, not named

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Records
- [x] API
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. ask.Collection, CollectionStore (memory and Postgres, kind collection: ids only, never expired, in the export, erased with the owner), ask.CollectionService (Create/Update/Delete/List, each document checked live as the reader, the rest counted in hidden; saved questions shown as a share is; stale after 90 days unasked), Request.Collection through withDocs with the three-chunk cap kept. /v1/collections. Web: Collections with Ask this collection. TestContextForTheWorkThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
