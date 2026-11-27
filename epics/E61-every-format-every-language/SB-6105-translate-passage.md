---
title: "SB-6105: Translate a cited passage"
summary: "A cited passage in a language the reader doesn't read is no help."
---

# SB-6105: Translate a cited passage

| Field | Value |
|---|---|
| Epic | [E61 Every Format, Every Language](README.md) |
| Wave / slice | W60 / S60.2 |
| Stories | US-61.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-26, LAW-28 |
| Depends on | SB-3001, SB-6103 |
| Status | Done |

## Why

A cited passage in a language the reader doesn't read is no help.

## Acceptance criteria

- [x] The passage view has **Translate** when the passage's language isn't the UI's
- [x] The gated small tier translates at most 600 characters, shown beside the original and marked
- [x] It is kept nowhere and checked as the reader; a reader who can't open the passage can't translate it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] `GET /v1/traces/{id}/passage?translate=`
- [x] Passage view button
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `GET /v1/traces/{id}/passage?translate=LANG` (`Agent.TranslatePassage`, the bundled `translate-passage` prompt) translates at most 600 characters of the passage the reader may open now, marked `read_by: translate`, kept nowhere; the web passage view's **Translate** shows it under the original. **Translate** is offered on every passage, as the cell doesn't know a passage's language. Tested in TestTranslatePassage.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
