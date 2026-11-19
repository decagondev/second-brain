---
title: "SB-1101: Template catalogue as data and sb space init"
summary: "A company should pick departments, not design them."
---

# SB-1101: Template catalogue as data and `sb space init`

| Field | Value |
|---|---|
| Epic | [E11 Department Templates and Connectors](README.md) |
| Wave / slice | W11 / S11.1 Catalogue |
| Stories | US-11.1 |
| Manual | `sb-departments(7)` (new), `sb-space(5)` (new), [`sb(1)`](../../manual/sb.1.md) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-1001](../E10-department-spaces/SB-1001-spaces-in-the-manifest-and-sb-tenant-check.md) |
| Status | Open |

## Why

A company should pick departments, not design them.

## Acceptance criteria

- [ ] Thirteen templates shipped in the release bundle as data, never code
- [ ] `sb space init --template people` writes a valid `spaces:` entry into the manifest
- [ ] A company can rename, merge or drop any template, and `custom` covers the rest

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Catalogue files
- [ ] CLI
- [ ] Bundle pinning
- [ ] Examples
- [ ] Write or extend the law tests: LAW-16
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
