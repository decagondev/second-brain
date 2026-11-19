---
title: "SB-015: Eval harness scaffold"
summary: "Evals become the main gate in W5; their harness must exist from W0 so suites accrue."
---

# SB-015: Eval harness scaffold

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W0 / S0.4 Laws red, models green |
| Stories | US-06.1 |
| Manual | [`sb-eval(1)`](../../manual/sb-eval.1.md), [`sb-eval(5)`](../../manual/sb-eval.5.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | — |
| Depends on | [SB-001](../E00-foundations/SB-001-repo-skeleton-and-toolchain.md) |
| Status | Done |

## Why

Evals become the main gate in W5; their harness must exist from W0 so suites accrue.

## Acceptance criteria

- [x] `sb eval run evals/smoke.yaml` runs against a mock model and prints a pass/fail table
- [x] Eval file format matches `sb-eval(5)`

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Python package `evals/` with a runner and YAML loader
- [x] Mock model adapter for deterministic runs
- [x] One smoke suite with three cases
- [x] CI job (non-blocking until W5)
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-18. `evals/` Python package: suite loader, the seven expectation checks from `sb-eval(5)`, the worst-of-three-runs rule, and a deterministic mock target. `sb eval run evals/suites/smoke.yaml --model mock` passes; `--tenant` and `--release` exit `ENOTIMPL` (SB-501). Unit tests cover every check passing and failing, unknown expectations, and the worst-of-three rule. It runs in `make check`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
