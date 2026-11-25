---
title: "SB-5002: Routing policies"
summary: "Every space pays the same for every task, whether it needs the largest model or not."
---

# SB-5002: Routing policies

| Field | Value |
|---|---|
| Epic | [E50 Run AI Like a Business](README.md) |
| Wave / slice | W49 / S49.1 |
| Stories | US-50.1 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | LAW-06, LAW-07 |
| Depends on | SB-4405, SB-3902 |
| Status | Done |

## Why

Every space pays the same for every task, whether it needs the largest model or not.

## Acceptance criteria

- [x] `routing:` (economy, balanced or frontier) per space, `balanced` by default, flows down the tree; each policy picks the documented tier per task
- [x] The gate runs each space's eval suite on its policy and refuses a policy that fails
- [x] A child can't choose a higher tier than its parent allows

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Setting and inheritance
- [x] Router
- [x] Gate check
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: `routing: economy|balanced|frontier` on a space (sb-space(5)), `balanced` by default, inherited by teams, and a team above its department is refused. `ask.policyTier` picks the tier: economy sends everything but research to the small tier, frontier everything to main, balanced keeps the SB-4405 rule; a held small tier answers on main whatever the policy. The cell has two tiers, so frontier is the main model. The evals job runs an economy space's cases on the small tier and a frontier space's on main only, so a failing case under its policy holds the small tier (the release gate's hold) rather than refusing the manifest. Tested in TestRoutingPoliciesAndLimits, TestSpaceRouting and TestSpendWithPurposeThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
