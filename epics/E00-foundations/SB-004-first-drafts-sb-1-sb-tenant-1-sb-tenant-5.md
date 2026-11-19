---
title: "SB-004: First drafts: sb(1), sb-tenant(1), sb-tenant(5)"
summary: "The tenant manifest and its three verbs are the product surface that portfolio IT touches."
---

# SB-004: First drafts: sb(1), sb-tenant(1), sb-tenant(5)

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.2 First pages |
| Stories | US-00.1 |
| Manual | [`sb(1)`](../../manual/sb.1.md), [`sb-tenant(1)`](../../manual/sb-tenant.1.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | — |
| Depends on | [SB-002](../E00-foundations/SB-002-manual-layout-man-page-template-and-docs.md) |
| Status | In review |

## Why

The tenant manifest and its three verbs are the product surface that portfolio IT touches.

## Acceptance criteria

- [ ] All three pages reviewed by two maintainers
- [x] Every manifest field has a type, default, example and validation rule
- [x] Exit codes are defined once and reused

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Draft `sb(1)` global options, exit codes and environment
- [x] Draft `sb-tenant(1)` for `check`, `apply`, `verify`
- [x] Draft `sb-tenant(5)` field table and examples
- [ ] Review and resolve comments
- [x] Devlog entry, ledger and changelog updated

## Progress

Drafts complete: `sb(1)` now specifies `ENOTIMPL` for commands that aren't built yet, plus every field and exit code. `tools/check/examples.py` checks every page has NAME, SYNOPSIS, DESCRIPTION and SEE ALSO in order.

**Left:** review by two maintainers.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
