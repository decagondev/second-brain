---
title: "SB-6106: Spanish, Italian and Dutch"
summary: "The UI and answers have only English, German and French."
---

# SB-6106: Spanish, Italian and Dutch

| Field | Value |
|---|---|
| Epic | [E61 Every Format, Every Language](README.md) |
| Wave / slice | W60 / S60.2 |
| Stories | US-61.2 |
| Manual | [`sb-tour(7)`](../../manual/sb-tour.7.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | none new |
| Depends on | SB-2301, SB-3003 |
| Status | Done |

## Why

The UI and answers have only English, German and French.

## Acceptance criteria

- [x] The web UI, chat replies and notices in Spanish, Italian and Dutch
- [x] Answers asked in each of them are in that language
- [x] The completeness test covers all six languages

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Strings in `web/src/i18n`
- [x] Chat and notice strings
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. Spanish, Italian and Dutch for all 1,163 strings of the web UI's catalog, with the same placeholders (i18n.test.ts checks all six languages and their plural rule); sb-tenant(5) `language` and the answer language (`ask.Languages`) take `es`, `it` and `nl`. The chat apps' fixed lines stay in English, as they are for German and French; their answers are in the manifest's language.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
