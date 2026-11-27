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
| Status | Done |

## Why

When an answer misses a document someone expected, nobody can tell why.

## Acceptance criteria

- [x] **Why not…?** takes a title or link and, for a document the person can open, says which applies: not indexed yet, outside the scope, ranked below the cut (with its rank and closest passage), skipped as suspect, held, dropped or excluded, or retired
- [x] Each reason has a fixture
- [x] A document the person can't open gets the same reply as one that doesn't exist

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Replay of retrieval as the person
- [x] Web: **Why not…?**
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `Agent.WhyNot` at `POST /v1/traces/{id}/why-not` resolves a title, link or KIND:ID (`resolveDoc`), answers `no_such_document` for one the caller can't open at its source exactly as for a missing one, and otherwise says `used`, `not_indexed`, `outside_scope`, `dropped`, `excluded`, `retired`, `held`, `suspect` or `ranked_below` with its rank, the cut and its closest passage (`Agent.ranked`, the trace's question replayed as the caller). Web: **Why not this document?** under each answer.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
