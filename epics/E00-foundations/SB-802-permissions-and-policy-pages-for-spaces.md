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
| Status | Open |

## Why

The access decision gains three checks before the source's ACL; the policy page must say exactly how they compose.

## Acceptance criteria

- [ ] The permission triple gains space, label and barrier context, and the order of checks is written down with a diagram
- [ ] The Cedar schema draft names `Space`, `Label` and `Barrier`, and the rule that a space policy can only forbid, never permit past the source
- [ ] New error codes (`ESPACE`, `EBARRIER`, `ELABEL`) are in the registry

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Rewrite the decision section of `sb-permissions(7)`
- [ ] Schema draft in `sb-policy(5)`
- [ ] Error codes in `sb-errors(7)` and `make gen`
- [ ] Write or extend the law tests: LAW-03, LAW-19, LAW-20, LAW-22
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
