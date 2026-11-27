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
| Status | Done |

## Why

A steward looking at a reported answer can't see what retrieval found and why it was cut.

## Acceptance criteria

- [x] On a reported or unhelpful answer, the space's steward sees the ranked passages: lexical and vector ranks, the fused score, and why each was kept or cut
- [x] It runs again as the steward; passages they can't open are counted
- [x] The asker's memory and exclusions are never shown

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Retrieval replay with ranks
- [x] Web: retrieval view on the review list
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `Stewards.Retrieval` at `GET /v1/spaces/NAME/review/retrieval` replays a reported or unhelpful answer's question as the steward (`Stewards.flagged`), every ranked candidate with its keyword and vector ranks, `retrieve.Score` and why it was kept or cut; ones the steward can't open are counted. It never runs as the asker, so their memory and exclusions play no part. Web: **Show the retrieval** on a reported answer.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
