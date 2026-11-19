---
title: "SB-016: Smoke and regression test conventions"
summary: "Regression and smoke tests stay forever, even once laws are the main gate."
---

# SB-016: Smoke and regression test conventions

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.1 Skeleton |
| Stories | US-00.5 |
| Manual | — |
| Laws | — |
| Depends on | [SB-001](../E00-foundations/SB-001-repo-skeleton-and-toolchain.md) |
| Status | Done |

## Why

Regression and smoke tests stay forever, even once laws are the main gate.

## Acceptance criteria

- [x] `tests/smoke/` and `tests/regress/` exist with one example each
- [x] Every bug in BUGS_AND_MITIGATIONS.md must link a regression test (DoD)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Create folders and naming convention `BUG-nnnn_*.go`
- [x] Add `make smoke` target
- [x] Document in docs/04-testing-and-assurance.md
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-18. `tests/smoke/` builds the real binary and checks the exit-status contract. `tests/regress/` holds the BUG-0003 test, plus a README mapping every bug to its test. `make smoke` and `make regress` are part of `make check`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
