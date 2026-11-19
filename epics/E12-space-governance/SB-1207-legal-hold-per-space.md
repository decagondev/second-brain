---
title: "SB-1207: Legal hold per space"
summary: "A legal hold must stop deletion in the scope of a matter, and only there."
---

# SB-1207: Legal hold per space

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.4 Harden |
| Stories | US-12.4 |
| Manual | `sb-spaces(7)` (new), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | [LAW-11](../../manual/sb-laws.7.md#law-11), [LAW-13](../../manual/sb-laws.7.md#law-13) |
| Depends on | [SB-611](../E07-rollout-ops/SB-611-retention-and-deletion.md) |
| Status | Open |

## Why

A legal hold must stop deletion in the scope of a matter, and only there.

## Acceptance criteria

- [ ] A Legal steward can place a hold on a space or a scope; retention and space deletion skip held items
- [ ] Holds and releases are audited; a held item can't be crypto-shredded

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Hold record
- [ ] Retention integration
- [ ] Audit rows
- [ ] Write or extend the law tests: LAW-11, LAW-13
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
