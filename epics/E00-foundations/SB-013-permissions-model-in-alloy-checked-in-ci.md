---
title: "SB-013: Permissions model in Alloy, checked in CI"
summary: "The permission rules are relational, which is exactly what Alloy is for."
---

# SB-013: Permissions model in Alloy, checked in CI

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.4 Laws red, models green |
| Stories | US-00.4 |
| Manual | [`sb-permissions(7)`](../../manual/sb-permissions.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-03](../../manual/sb-laws.7.md#law-03) |
| Depends on | [SB-001](../E00-foundations/SB-001-repo-skeleton-and-toolchain.md) |
| Status | In review |

## Why

The permission rules are relational, which is exactly what Alloy is for.

## Acceptance criteria

- [x] `proofs/alloy/Permissions.als` checks all assertions with no counterexample at scope 6
- [x] Ledger: model evidence recorded for LAW-01, LAW-02 and LAW-03 (they stay Tested (red), which ranks above Modelled)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Add Alloy CLI job (pinned jar, checksum-verified)
- [x] Mutation-check the `unknown` rule
- [x] Record in devlog
- [x] Write or extend the law tests: LAW-01, LAW-02, LAW-03
- [x] Devlog entry, ledger and changelog updated

## Progress

CI job wired with `alloy.jar` 6.2.0, checksum-pinned. `proofs.sh` only passes the model if every `check` is reported UNSAT; any other output fails.

2026-11-19: first real run. The planned flags were wrong: Alloy 6.2 `exec` runs only the default command, and on Windows the Java launcher expands a `*` argument. `proofs.sh` now runs each `check` by name (`-c NAME -t none`), needs `UNSAT` for each, and needs the `run SomeCitedAnswer` to be `SAT` so the model isn't vacuous. All 5 checks are UNSAT at scope 6. Mutation `context = candidates - live.Deny` makes `UnknownNeverUsed`, `MissingNeverUsed` and `CitedImpliesLiveAllow` SAT, as expected.

2026-11-19: first CI run green on `c0a12be` (all jobs: check, proofs, postgres, diagrams).

**Left:** two reviewers (proofs are a two-reviewer area, DoD).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
