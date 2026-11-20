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
| Status | In review |

## Why

Before any company relies on HR or Finance spaces, someone should try hard to break them.

## Acceptance criteria

- [x] A red-team plan with mislabelled files, over-broad scopes, injected 'ask the HR brain' instructions and mid-reorg asks
- [ ] A game day on staging with a friendly tenant (human); findings in the register (runbook ready; the game day is human)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Plan
- [x] Tooling and corpus
- [x] Game day runbook
- [x] Write or extend the law tests: LAW-19, LAW-20, LAW-21, LAW-22
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S12.4b): the plan is [docs/red-team-spaces.md](../../docs/red-team-spaces.md), with nine attacks and how each is automated, and the game day runbook is [docs/runbooks/space-red-team-game-day.md](../../docs/runbooks/space-red-team-game-day.md). sb-laws(7) and sb-eval(5) name the new `space-redteam` suite: a world of source-side mistakes (a case file mislabelled `confidential`, a restricted file misfiled outside every space, a planted "as the HR brain" instruction, a deal file shared by its owner across a barrier, a person moved out of People) with six attacks and two controls, all passing and in the release gate. The evalcell world gained a document `owner`. Attack 3, an over-broad scope, worked on paper: a space scoped `/**` or `/People/Cases/**` beside a restricted People space made its other members holders of People's files. That is PT-0003 (BUG-0013), fixed by sb-space(5) rule 13 in `sb tenant check`, with `TestScopeReach` and the law test `TestLAW20_NoScopeReachesARestrictedSpace` (a check without the rule is caught). PT-0004 records the accepted design risk of two source mistakes at once.

**Left:** the game day itself, on staging with a friendly tenant (human).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
