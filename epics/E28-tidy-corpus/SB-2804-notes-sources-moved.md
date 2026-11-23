---
title: "SB-2804: Notes whose sources moved"
summary: "A note keeps answering after the documents it rests on have changed."
---

# SB-2804: Notes whose sources moved

| Field | Value |
|---|---|
| Epic | [E28 A Tidy Corpus](README.md) |
| Wave / slice | W27 / S27.1 |
| Stories | US-28.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Open |

## Why

A note keeps answering after the documents it rests on have changed.

## Acceptance criteria

- [ ] A note records the versions it cites when published; a new version or a gone document puts it back on the review list
- [ ] An edit or a review mark clears it; the changed source is named only if the steward can open it
- [ ] A new version of a cited document lists the note; a mark clears it

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Record
- [ ] Review list
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
