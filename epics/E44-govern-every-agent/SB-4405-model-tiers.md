---
title: "SB-4405: Model tiers"
summary: "Every ask costs the same, though most are short and plain."
---

# SB-4405: Model tiers

| Field | Value |
|---|---|
| Epic | [E44 Govern Every Agent](README.md) |
| Wave / slice | W43 / S43.2 |
| Stories | US-44.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | LAW-06, LAW-07 |
| Depends on | SB-305 |
| Status | Done |

## Why

Every ask costs the same, though most are short and plain.

## Acceptance criteria

- [x] `model.small` takes asks with no agent, no file and a short question; the rest use the main model
- [x] The release gate runs the eval suites on both tiers, and a failing tier takes no asks; a space can pin `model: main`
- [x] The costs page shows spend per tier, and the report the share of asks on each

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Config
- [x] Routing
- [x] Gate
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. sb-cell(5) `model.small.id` and its price, the same provider, base and key. Agent.tierFor sends an ask with no agent, file, chosen documents or collection, a question of at most 120 characters and no earlier turn, in a space that doesn't pin sb-space(5) `model: main`, to the small model while the small tier isn't held; the trace records its tier. `make evals` runs every suite for rel_local and rel_local_small and gates both; the evals job runs the tenant's cases on both tiers and holds the small tier (state/evals/small-held) when one passes on main and fails on small. The cost statement names each line's tier and the weekly report the share of asks on the small model. TestTierFor and the cell test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
