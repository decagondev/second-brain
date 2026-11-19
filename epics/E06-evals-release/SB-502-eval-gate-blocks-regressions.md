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
| Status | Open |

## Why

The gate is the policy; humans can't wave a release through.

## Acceptance criteria

- [ ] A seeded regression blocks promotion
- [ ] Thresholds live in `evals/gate.yaml` and changes need two reviewers

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Gate step
- [ ] Threshold file
- [ ] Seeded regression test
- [ ] Write or extend the law tests: LAW-06
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
