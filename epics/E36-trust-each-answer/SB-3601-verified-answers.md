---
title: "SB-3601: Verified answers"
summary: "Stewards answer the same question well once, but the next asker gets a fresh model answer."
---

# SB-3601: Verified answers

| Field | Value |
|---|---|
| Epic | [E36 Trust Each Answer](README.md) |
| Wave / slice | W35 / S35.1 |
| Stories | US-36.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-01, LAW-14, LAW-19 |
| Depends on | SB-3402 |
| Status | Done |

## Why

Stewards answer the same question well once, but the next asker gets a fresh model answer.

## Acceptance criteria

- [x] A space's stewards pin an answer to its question, at most 50 a space, one audited record each
- [x] A close new ask in the space shows the pin first, as Verified by the stewards, with every cite checked again as the asker
- [x] A pin with a cite the asker can't open, or from another space, is never shown to them

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Store and API
- [x] Agent
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Pins of cited steward replies, shown first with every cite checked as the asker

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
