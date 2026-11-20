---
title: "SB-1203: Information barriers"
summary: "Deal teams, investigations and some legal matters must not see, or be seen by, the rest of the company."
---

# SB-1203: Information barriers

| Field | Value |
|---|---|
| Epic | [E12 Space Governance](README.md) |
| Wave / slice | W12 / S12.2 Barriers and duties |
| Stories | US-12.2 |
| Manual | [`sb-permissions(7)`](../../manual/sb-permissions.7.md), `sb-space(5)` (new) |
| Laws | LAW-22 (new, W8) |
| Depends on | [SB-1004](../E10-department-spaces/SB-1004-cedar-schema-v2-spaces-labels-narrowing.md) |
| Status | Done |

## Why

Deal teams, investigations and some legal matters must not see, or be seen by, the rest of the company.

## Acceptance criteria

- [x] `barriers:` pairs of groups become Cedar forbids in both directions, for reads, drafts and actions
- [x] A document owned by a barred group is invisible across the barrier, even company-wide and even if shared in the source
- [x] LAW-22 green

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Policy generation
- [x] Ownership from the source
- [x] LAW-22 law test green
- [x] Write or extend the law tests: LAW-22
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S12.2a): sb-permissions(7) gained "Barriers beyond reads, exactly" and the owner in step 3, sb-policy(5) the new facts and forbids, sb-space(5) and sb-spaces(7) what a barrier covers, and sb-connector(3) the source owner. Schema: `Chunk.owners`, `barred_groups` on reads, target facts on `call_tool` and `propose`, and `requester_groups` on `approve`. `spaces.cedar` gained forbids for content owned across a barrier, for tool calls and proposals on an item held or owned across one (or one the cell can't place), and for approvals across one; rule 0 now also needs `owners` and `barred_groups`. Documents and chunks carry `Owner` (Drive: the file's owner; the postgres index has an `owner` column); retrieval resolves the owner's groups now (`Authoriser.Owners`, `barrier.Directory`), and an owner it can't resolve refuses. `Authoriser.ItemAllowed` puts a draft's item through routing, the space steps and the live ACL; `internal/core/barrier` gives the tool bus the target and approval facts (`Bus.Barriers`). `TestLAW22_BarriersHold` now runs asks, drafts, tool calls and approvals in random companies with owners through the real code and policies, and is green and out of the ratchet; a cell that ignores owners, or gives the bus no target or approval facts, is caught, and the test fails if no draft, action or approval is ever allowed.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
