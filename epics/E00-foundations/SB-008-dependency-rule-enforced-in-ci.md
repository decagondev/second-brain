---
title: "SB-008: Dependency rule enforced in CI"
summary: "SOLID only survives if `core` cannot import an adapter."
---

# SB-008: Dependency rule enforced in CI

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.3 Rules in CI |
| Stories | — |
| Manual | — |
| Laws | — |
| Depends on | [SB-001](../E00-foundations/SB-001-repo-skeleton-and-toolchain.md) |
| Status | Done |

## Why

SOLID only survives if `core` cannot import an adapter.

## Acceptance criteria

- [x] `internal/core/**` cannot import `internal/adapters/**` or any vendor SDK; CI fails if it does
- [x] The allowed import graph in docs/architecture/00-overview.md is the lint config

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Configure `depguard` rules from the layering table
- [x] Add a deliberate violation test and watch it fail
- [x] Document the rule in the engineering standards
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-18. Enforced twice: `depguard` rules in `.golangci.yml`, and `tools/check/deps.py`, which reads `go list` directly. Mutation-checked: a planted import of `internal/errs` from `internal/core/ports` was caught by both.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
