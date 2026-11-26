---
title: "SB-5506: Why not this document?"
summary: "When an answer misses a document someone expected, nobody can tell why."
---

# SB-5506: Why not this document?

| Field | Value |
|---|---|
| Epic | [E55 See and Steer the Context](README.md) |
| Wave / slice | W54 / S54.2 |
| Stories | US-55.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-01, LAW-09 |
| Depends on | SB-5501, SB-5101, SB-5402 |
| Status | Open |

## Why

When an answer misses a document someone expected, nobody can tell why.

## Acceptance criteria

- [ ] **Why not…?** takes a title or link and, for a document the person can open, says which applies: not indexed yet, outside the scope, ranked below the cut (with its rank and closest passage), skipped as suspect, held, dropped or excluded, or retired
- [ ] Each reason has a fixture
- [ ] A document the person can't open gets the same reply as one that doesn't exist

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Replay of retrieval as the person
- [ ] Web: **Why not…?**
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
