---
title: "SB-502: Eval gate blocks regressions"
summary: "The gate is the policy; humans can't wave a release through."
---

# SB-502: Eval gate blocks regressions

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W5 / S5.1 Eval gate |
| Stories | US-06.1 |
| Manual | [`sb-release(1)`](../../manual/sb-release.1.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | [LAW-06](../../manual/sb-laws.7.md#law-06) |
| Depends on | [SB-501](../E06-evals-release/SB-501-eval-runner-and-suites.md) |
| Status | In review |

## Why

The gate is the policy; humans can't wave a release through.

## Acceptance criteria

- [x] A seeded regression blocks promotion
- [ ] Thresholds live in `evals/gate.yaml` and changes need two reviewers (the file is pinned in the bundle by hash; two-reviewer enforcement needs branch protection and CODEOWNERS set by a repository admin)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Gate step
- [x] Threshold file
- [x] Seeded regression test
- [x] Write or extend the law tests: LAW-06
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S5.1b): `python3 -m evals gate RESULTS/REL [--stable RESULTS/REL]` applies `evals/gate.yaml`: `failures_max` on failures, and `score_min` on mean support for `citation` and on pass rate for everything else, plus `drop_max` against stable. A missing suite fails. It prints a line per suite and exits 1 (`EEVAL`). Seeded regressions are tested, and each one blocks: a permission failure, citation under the floor, a citation drop against stable, a refusal-rate miss, and a missing suite. `make evals` gates `rel_local` on every `make check`. The gate file's hash is in the release bundle (SB-503), so loosening a threshold is a new release id.

**Left:** branch protection and CODEOWNERS for `evals/gate.yaml` (a repository admin), and wiring the gate into `sb release promote` (SB-507).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
