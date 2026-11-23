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
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-permissions(7)`](../../manual/sb-permissions.7.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Done |

## Why

Two equally relevant documents are ordered by chance, not by age.

## Acceptance criteria

- [x] Among chunks with the same fused score, the newer version comes first; nothing else in ranking changes
- [x] It changes only the order, never what passes the checks
- [x] A tie goes to the newer document; the eval suite's scores don't move

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Ranking
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S26.2; tests green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
