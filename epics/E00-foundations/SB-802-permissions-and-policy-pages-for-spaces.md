---
title: "SB-802: sb-permissions(7) and sb-policy(5) for spaces, labels and barriers"
summary: "The access decision gains three checks before the source's ACL; the policy page must say exactly how they compose."
---

# SB-802: `sb-permissions(7)` and `sb-policy(5)` for spaces, labels and barriers

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W8 / S8.1 Space pages |
| Stories | US-00.6 |
| Manual | [`sb-permissions(7)`](../../manual/sb-permissions.7.md), [`sb-policy(5)`](../../manual/sb-policy.5.md), [`sb-errors(7)`](../../manual/sb-errors.7.md) |
| Laws | [LAW-03](../../manual/sb-laws.7.md#law-03), LAW-19 (new, W8), LAW-20 (new, W8), LAW-22 (new, W8) |
| Depends on | [SB-801](SB-801-sb-spaces-7-and-sb-space-5-first-drafts.md) |
| Status | Done |

## Why

The access decision gains three checks before the source's ACL; the policy page must say exactly how they compose.

## Acceptance criteria

- [x] The permission triple gains space, label and barrier context, and the order of checks is written down with a diagram
- [x] The Cedar schema draft names `Space`, `Label` and `Barrier`, and the rule that a space policy can only forbid, never permit past the source
- [x] New error codes (`ESPACE`, `EBARRIER`, `ELABEL`) are in the registry

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Rewrite the decision section of `sb-permissions(7)`
- [x] Schema draft in `sb-policy(5)`
- [x] Error codes in `sb-errors(7)` and `make gen`
- [x] Write or extend the law tests: LAW-03, LAW-19, LAW-20, LAW-22 (red tests in SB-804)
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S8.1): sb-permissions(7) gained "Spaces, labels and barriers, exactly". It has the four checks in order with a diagram, membership (`ESPACE`), label (`ELABEL`) and barrier (`EBARRIER`) before the source's live ACL, which stays last. A space can only forbid (LAW-19), and any `Unknown` refuses. The new codes are reasons, never shown with a chunk. sb-policy(5) gained the schema v2 draft (`Space`, `Label`, `Barrier`, `Chunk.label`, `Chunk.spaces`, `context.space`) and the three rules: only forbids, barriers generated from the manifest, and a missing attribute refuses. sb-errors(7) has the three codes, and `make gen` wrote them into `internal/errs`. The law tests for LAW-03, LAW-19, LAW-20 and LAW-22 over spaces are SB-804's red tests.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
