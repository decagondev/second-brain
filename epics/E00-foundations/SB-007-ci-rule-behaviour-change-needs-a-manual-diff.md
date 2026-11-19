---
title: "SB-007: CI rule: behaviour change needs a manual diff"
summary: "P2 is only real if the build enforces it."
---

# SB-007: CI rule: behaviour change needs a manual diff

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.3 Rules in CI |
| Stories | US-00.2 |
| Manual | [`sb(1)`](../../manual/sb.1.md) |
| Laws | — |
| Depends on | [SB-001](../E00-foundations/SB-001-repo-skeleton-and-toolchain.md), [SB-002](../E00-foundations/SB-002-manual-layout-man-page-template-and-docs.md) |
| Status | Done |

## Why

P2 is only real if the build enforces it.

## Acceptance criteria

- [x] A PR touching `cmd/` or `internal/` without touching `manual/` fails
- [x] Label `refactor-no-behaviour` overrides, and the override is logged in the PR
- [x] The rule is mutation-tested once (devlog records it)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Write `tools/ci/manual_diff.sh`
- [x] Add CI job and label override
- [x] Open a test PR that must fail; record result
- [x] Document in CONTRIBUTING.md
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-18. `tools/ci/manual_diff.sh` runs in CI on every PR. `tools/ci/test_manual_diff.sh` proves the rule in a throwaway git repo: code-only fails, code plus manual passes, the override label passes, docs-only passes. It runs in `make check` (`ci-rules`).

The planned "open a test PR" was replaced by the scripted test, which runs on every build rather than once. The first real PR should still be watched.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
