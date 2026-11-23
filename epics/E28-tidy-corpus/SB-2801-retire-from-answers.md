---
title: "SB-2801: Retire from answers"
summary: "Nothing takes a superseded document out of answers short of deleting it at the source."
---

# SB-2801: Retire from answers

| Field | Value |
|---|---|
| Epic | [E28 A Tidy Corpus](README.md) |
| Wave / slice | W27 / S27.1 |
| Stories | US-28.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-19, LAW-11, LAW-13 |
| Depends on | — |
| Status | Open |

## Why

Nothing takes a superseded document out of answers short of deleting it at the source.

## Acceptance criteria

- [ ] A steward retires a document from their space's answers, with a reason and an optional replacement; answers, searches and drafts in that space skip it
- [ ] Bring back undoes it; each is a record and an audit row; the replacement's cite says what it replaces if the asker can open the old one
- [ ] A retired document is never cited in that space and still is in another; a rebuild keeps it out

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API and record
- [ ] Retrieval
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
