---
title: "SB-2706: Newer wins a tie"
summary: "Two equally relevant documents are ordered by chance, not by age."
---

# SB-2706: Newer wins a tie

| Field | Value |
|---|---|
| Epic | [E27 Fresh Answers](README.md) |
| Wave / slice | W26 / S26.2 |
| Stories | US-27.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Open |

## Why

Two equally relevant documents are ordered by chance, not by age.

## Acceptance criteria

- [ ] Among chunks with the same fused score to three decimals, the newer version comes first; nothing else in ranking changes
- [ ] Ordering runs after the checks
- [ ] A tie goes to the newer document; the eval suite's scores don't move

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Ranking
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
