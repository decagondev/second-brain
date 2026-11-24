---
title: "SB-3806: Search from the address bar"
summary: "Searching the brain means opening it first."
---

# SB-3806: Search from the address bar

| Field | Value |
|---|---|
| Epic | [E38 Work Where People Are](README.md) |
| Wave / slice | W37 / S37.2 |
| Stories | US-38.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | — |
| Depends on | SB-1701 |
| Status | Done |

## Why

Searching the brain means opening it first.

## Acceptance criteria

- [x] The cell serves `/opensearch.xml` pointing at the Documents tab with the words as q
- [x] The search runs signed in as always
- [x] The file is valid OpenSearch

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Cell
- [x] UI
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `api.Web.Base` and `Web.openSearch`: `GET /opensearch.xml` (OpenSearch 1.1, `BASE/?tab=documents&q={searchTerms}`), linked from index.html; `App` takes `q` and runs it on the Documents tab. TestOpenSearch (parsed as XML), web brief.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
