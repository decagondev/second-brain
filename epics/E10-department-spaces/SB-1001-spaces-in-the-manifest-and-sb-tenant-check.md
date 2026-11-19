---
title: "SB-1001: Spaces in the manifest and sb tenant check"
summary: "Spaces are configuration; the manifest stays the truth."
---

# SB-1001: Spaces in the manifest and `sb tenant check`

| Field | Value |
|---|---|
| Epic | [E10 Department Spaces](README.md) |
| Wave / slice | W10 / S10.1 Space model |
| Stories | US-10.1 |
| Manual | `sb-space(5)` (new), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-801](../E00-foundations/SB-801-sb-spaces-7-and-sb-space-5-first-drafts.md) |
| Status | Open |

## Why

Spaces are configuration; the manifest stays the truth.

## Acceptance criteria

- [ ] The parser accepts `spaces:` and `barriers:` exactly as `sb-space(5)` says, with line-precise errors
- [ ] Rules: unknown template, member or steward groups not in `group:` form, a scope naming a connector the tenant lacks, and a restricted space with no steward are all errors
- [ ] `sb tenant apply --plan` shows space changes

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Schema and parser
- [ ] Validation rules
- [ ] Plan output
- [ ] Examples in `examples/tenants`
- [ ] Write or extend the law tests: LAW-16
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
