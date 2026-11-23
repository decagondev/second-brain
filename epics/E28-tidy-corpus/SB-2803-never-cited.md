---
title: "SB-2803: Never cited"
summary: "Nobody sees which documents answers never use."
---

# SB-2803: Never cited

| Field | Value |
|---|---|
| Epic | [E28 A Tidy Corpus](README.md) |
| Wave / slice | W27 / S27.1 |
| Stories | US-28.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | — |
| Status | Open |

## Why

Nobody sees which documents answers never use.

## Acceptance criteria

- [ ] `GET /v1/spaces/NAME/review?section=uncited` lists documents no answer cited in 180 days and older than 90 days, at most 100, oldest first
- [ ] Titles only for documents the steward can open; never question text
- [ ] A document cited once in the window isn't listed; a hidden one is a count only

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
