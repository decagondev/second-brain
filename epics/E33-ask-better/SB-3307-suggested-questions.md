---
title: "SB-3307: Suggested questions"
summary: "A new person facing an empty ask box doesn't know what to ask in a space."
---

# SB-3307: Suggested questions

| Field | Value |
|---|---|
| Epic | [E33 Ask Better](README.md) |
| Wave / slice | W32 / S32.2 |
| Stories | US-33.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-11, LAW-19 |
| Depends on | SB-2503 |
| Status | Open |

## Why

A new person facing an empty ask box doesn't know what to ask in a space.

## Acceptance criteria

- [ ] A space's stewards keep up to 8 starter questions of at most 120 characters, one record per space, audited
- [ ] An empty ask box in that space shows them as buttons for its members; a company-wide ask shows none
- [ ] A ninth question or a non-steward gets refused

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Store and API
- [ ] Steward screen and UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
