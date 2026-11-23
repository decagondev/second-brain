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
| Status | Done |

## Why

Nothing takes a superseded document out of answers short of deleting it at the source.

## Acceptance criteria

- [x] A steward retires a document from their space's answers, with a reason and an optional replacement; answers, searches and drafts in that space skip it
- [x] Bring back undoes it; each is a record and an audit row; the replacement's cite says what it replaces if the asker can open the old one
- [x] A retired document is never cited in that space and still is in another; a rebuild keeps it out

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API and record
- [x] Retrieval
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S27.1; tested in internal/core/ask/tidy_test.go, tests/cell/retire_test.go and web/src/tidy.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
