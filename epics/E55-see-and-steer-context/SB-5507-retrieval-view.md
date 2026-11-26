---
title: "SB-5507: Retrieval view for stewards"
summary: "A steward looking at a reported answer can't see what retrieval found and why it was cut."
---

# SB-5507: Retrieval view for stewards

| Field | Value |
|---|---|
| Epic | [E55 See and Steer the Context](README.md) |
| Wave / slice | W54 / S54.2 |
| Stories | US-55.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-01, LAW-26 |
| Depends on | SB-3604, SB-3606 |
| Status | Open |

## Why

A steward looking at a reported answer can't see what retrieval found and why it was cut.

## Acceptance criteria

- [ ] On a reported or unhelpful answer, the space's steward sees the ranked passages: lexical and vector ranks, the fused score, and why each was kept or cut
- [ ] It runs again as the steward; passages they can't open are counted
- [ ] The asker's memory and exclusions are never shown

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Retrieval replay with ranks
- [ ] Web: retrieval view on the review list
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
