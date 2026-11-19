---
title: "SB-801: First drafts: sb-spaces(7) and sb-space(5)"
summary: "Every space ticket needs one page that says what a space is and one that says how to declare it."
---

# SB-801: First drafts: `sb-spaces(7)` and `sb-space(5)`

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W8 / S8.1 Space pages |
| Stories | US-00.6 |
| Manual | `sb-spaces(7)` (new), `sb-space(5)` (new), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16), LAW-19 (new, W8) |
| Depends on | — |
| Status | Done |

## Why

Every space ticket needs one page that says what a space is and one that says how to declare it.

## Acceptance criteria

- [x] `sb-spaces(7)` covers what a space is, membership, scopes, labels, stewards, company-wide asks and the access decision order
- [x] `sb-space(5)` defines the `spaces:` and `barriers:` manifest blocks with types, defaults and validation rules, and every example is a `yaml run` block
- [x] `sb-tenant(5)` links to it, and a manifest with no `spaces:` block behaves exactly as today

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Draft `sb-spaces(7)`
- [x] Draft `sb-space(5)` with examples
- [x] Cross-links from `sb-tenant(5)` and `sb-tenancy(7)`
- [x] Write or extend the law tests: LAW-16, LAW-19 (LAW-16 here; LAW-19's red test is SB-804's)
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S8.1): `sb-spaces(7)` covers what a space is, labels, barriers, company-wide asks, the four checks in order, stewards and moves. `sb-space(5)` defines the `spaces:` and `barriers:` blocks with types, defaults and six validation rules, and its examples are `yaml run` blocks that CI parses. sb-tenant(5) has both fields, marked refused until SB-1001, and sb-tenancy(7) and the manual index link to the pages. `sb tenant check` refuses either block by name (`not built yet (SB-1001)`), so a company can't believe a space protects what nothing enforces yet. A manifest without them is unchanged, and every existing test still passes. The new law test `TestLAW16_UnbuiltFieldsAreRefused` puts a spaces or barriers block anywhere in random valid manifests, and a self-test catches a parser that quietly drops one. The LAW-19 test is SB-804's.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
