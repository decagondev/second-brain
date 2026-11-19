---
title: "SB-012: Rollout model in TLA+, checked in CI"
summary: "Channel and ring rules interact with rollback in ways that are easy to get wrong."
---

# SB-012: Rollout model in TLA+, checked in CI

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.4 Laws red, models green |
| Stories | US-00.4 |
| Manual | [`sb-release(7)`](../../manual/sb-release.7.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06), [LAW-08](../../manual/sb-laws.7.md#law-08) |
| Depends on | [SB-001](../E00-foundations/SB-001-repo-skeleton-and-toolchain.md) |
| Status | In review |

## Why

Channel and ring rules interact with rollback in ways that are easy to get wrong.

## Acceptance criteria

- [x] `proofs/tla/Rollout.tla` model-checks with no violation
- [x] Dropping the promote guard yields a LAW-08 counterexample
- [x] Ledger: LAW-06 and LAW-08 move to Modelled

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Add TLC job for Rollout
- [x] Mutation-check the promote guard
- [x] Record in devlog
- [ ] Write or extend the law tests: LAW-06, LAW-08
- [x] Devlog entry, ledger and changelog updated

## Progress

`Rollout.tla` now models non-atomic rollback (BUG-0001 fixed in the model): revoked cells drain, then redeploy. The Python cross-check passes (6,290 states), and 4 mutations are caught, including dropping the promote guard.

2026-11-19: `make proofs` ran locally: TLC reports no error, 6,290 distinct states (same as the cross-check). All 4 mutations are caught: no promote guard gives `OneBehindMax`, ungated promote gives `OnlyGatedRuns`, and deploying a revoked release or rolling back without draining both give `RevokedOnlyWhileDraining`. LAW-06 and LAW-08 are Modelled. The law-test subtask stays open for SB-506 and SB-507.

2026-11-19: first CI run green on `c0a12be` (all jobs: check, proofs, postgres, diagrams).

**Left:** two reviewers (proofs are a two-reviewer area, DoD).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
