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
| Status | Open |

## Why

A steward can't see a bad answer unless someone retypes it.

## Acceptance criteria

- [ ] Under an unhelpful rating the asker can Send to the stewards, with an optional note; it joins the space's queue as a report
- [ ] The steward sees the question, the answer, and each document by title only if they can open it, the rest counted
- [ ] Nothing is sent without the asker's click; audited

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] Steward screen and UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
