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
| Status | Open |

## Why

The UI and answers have only English, German and French.

## Acceptance criteria

- [ ] The web UI, chat replies and notices in Spanish, Italian and Dutch
- [ ] Answers asked in each of them are in that language
- [ ] The completeness test covers all six languages

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Strings in `web/src/i18n`
- [ ] Chat and notice strings
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
