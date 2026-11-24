---
title: "SB-3606: Report this answer"
summary: "A steward can't see a bad answer unless someone retypes it."
---

# SB-3606: Report this answer

| Field | Value |
|---|---|
| Epic | [E36 Trust Each Answer](README.md) |
| Wave / slice | W35 / S35.2 |
| Stories | US-36.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07, LAW-11 |
| Depends on | SB-1704 |
| Status | Done |

## Why

A steward can't see a bad answer unless someone retypes it.

## Acceptance criteria

- [x] Under an unhelpful rating the asker can Send to the stewards, with an optional note; it joins the space's queue as a report
- [x] The steward sees the question, the answer, and each document by title only if they can open it, the rest counted
- [x] Nothing is sent without the asker's click; audited

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] Steward screen and UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Report this answer: Send to the stewards after Not helpful, by the asker's click, named; the steward sees only what they can open

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
