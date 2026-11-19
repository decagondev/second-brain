---
title: "SB-1208: Cross-space red team and game day"
summary: "Before any company relies on HR or Finance spaces, someone should try hard to break them."
---

# SB-1208: Cross-space red team and game day

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.4 Harden |
| Stories | US-12.5 |
| Manual | [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-19 (new, W8), LAW-20 (new, W8), LAW-21 (new, W8), LAW-22 (new, W8) |
| Depends on | [SB-1203](SB-1203-information-barriers.md), [SB-1205](SB-1205-scim-attributes-and-five-minute-moves.md) |
| Status | Open |

## Why

Before any company relies on HR or Finance spaces, someone should try hard to break them.

## Acceptance criteria

- [ ] A red-team plan with mislabelled files, over-broad scopes, injected 'ask the HR brain' instructions and mid-reorg asks
- [ ] A game day on staging with a friendly tenant (human); findings in the register

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Plan
- [ ] Tooling and corpus
- [ ] Game day runbook
- [ ] Write or extend the law tests: LAW-19, LAW-20, LAW-21, LAW-22
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
