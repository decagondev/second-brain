---
title: "SB-3001: Show the passage"
summary: "A cite gives a title and an age, but not the sentences it rests on."
---

# SB-3001: Show the passage

| Field | Value |
|---|---|
| Epic | [E30 Show the Evidence](README.md) |
| Wave / slice | W29 / S29.1 |
| Stories | US-30.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-01, LAW-07, LAW-09 |
| Depends on | — |
| Status | Open |

## Why

A cite gives a title and an age, but not the sentences it rests on.

## Acceptance criteria

- [ ] `GET /v1/traces/{id}/passage?cite=` checks the asker's access again and returns the cited span's text at that version, at most 600 characters cut at a sentence
- [ ] The UI shows it as a quote, never as markup; a source the asker can't open now says so and shows nothing
- [ ] Revoking access after the answer refuses the passage; HTML in a source shows as text; axe

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
