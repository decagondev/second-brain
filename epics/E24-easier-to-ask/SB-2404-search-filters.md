---
title: "SB-2404: Search filters"
summary: "Search can't be narrowed to a kind of file or to recent documents."
---

# SB-2404: Search filters

| Field | Value |
|---|---|
| Epic | [E24 Easier to Ask](README.md) |
| Wave / slice | W23 / S23.2 |
| Stories | US-24.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Done |

## Why

Search can't be narrowed to a kind of file or to recent documents.

## Acceptance criteria

- [x] `GET /v1/search` takes `type=` (from the extension on the source id or title; none is `other`) and `since=Nd` (the version's first-seen time)
- [x] The Documents tab shows both as drop-downs; filters run after the live ACL check
- [x] Each filter narrows; a document the person can't open never appears, whatever the filters

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API filters
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-23 (S23.2). `GET /v1/search` takes `type=` (pdf, doc, sheet, slides, web, text, other; `ask.TypeOf` reads the extension on the source id, else the title) and `since=Nd` (1 to 365, by the version's first-seen time; unknown age is left out). Both run in `ask.Agent.Search` after every check, so they only narrow and `refused` is unchanged. The Documents tab has **File type** and **Changed in**. Tests: `TestSearchFilters` (a denied document never appears under any filter, `refused` the same), `TestSearchFiltersEndpoint`, `findfaster.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
