---
title: "SB-1206: Reorg preview: who gains and loses before it happens"
summary: "Reorgs are planned; HR and IT should see the access effect before the IdP changes."
---

# SB-1206: Reorg preview: who gains and loses before it happens

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.3 Reorgs |
| Stories | US-12.3 |
| Manual | [`sb-tenant(1)`](../../manual/sb-tenant.1.md), `sb-spaces(7)` (new) |
| Laws | [LAW-16](../../manual/sb-laws.7.md#law-16) |
| Depends on | [SB-1205](SB-1205-scim-attributes-and-five-minute-moves.md) |
| Status | Open |

## Why

Reorgs are planned; HR and IT should see the access effect before the IdP changes.

## Acceptance criteria

- [ ] `sb space diff --idp-snapshot FILE` lists, per person, the spaces they'd gain and lose
- [ ] Flags anyone who would lose all spaces, or gain a restricted one
- [ ] Works offline on a snapshot; no change is made

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] CLI
- [ ] Snapshot format
- [ ] Flags
- [ ] Write or extend the law tests: LAW-16
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
