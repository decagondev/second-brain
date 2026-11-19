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
| Status | Open |

## Why

Every space ticket needs one page that says what a space is and one that says how to declare it.

## Acceptance criteria

- [ ] `sb-spaces(7)` covers what a space is, membership, scopes, labels, stewards, company-wide asks and the access decision order
- [ ] `sb-space(5)` defines the `spaces:` and `barriers:` manifest blocks with types, defaults and validation rules, and every example is a `yaml run` block
- [ ] `sb-tenant(5)` links to it, and a manifest with no `spaces:` block behaves exactly as today

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Draft `sb-spaces(7)`
- [ ] Draft `sb-space(5)` with examples
- [ ] Cross-links from `sb-tenant(5)` and `sb-tenancy(7)`
- [ ] Write or extend the law tests: LAW-16, LAW-19
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
