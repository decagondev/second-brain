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
| Status | Open |

## Why

A cited passage in a language the reader doesn't read is no help.

## Acceptance criteria

- [ ] The passage view has **Translate** when the passage's language isn't the UI's
- [ ] The gated small tier translates at most 600 characters, shown beside the original and marked
- [ ] It is kept nowhere and checked as the reader; a reader who can't open the passage can't translate it

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] `GET /v1/traces/{id}/passage?translate=`
- [ ] Passage view button
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
