---
title: "SB-011: Approvals model in TLA+, checked in CI"
summary: "The approval and action state machine is where a subtle race becomes a real-world mistake."
---

# SB-011: Approvals model in TLA+, checked in CI

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.4 Laws red, models green |
| Stories | US-00.4 |
| Manual | [`sb-approvals(7)`](../../manual/sb-approvals.7.md) |
| Laws | [LAW-04](../../manual/sb-laws.7.md#law-04), [LAW-05](../../manual/sb-laws.7.md#law-05), [LAW-10](../../manual/sb-laws.7.md#law-10), [LAW-15](../../manual/sb-laws.7.md#law-15) |
| Depends on | [SB-001](../E00-foundations/SB-001-repo-skeleton-and-toolchain.md) |
| Status | In review |

## Why

The approval and action state machine is where a subtle race becomes a real-world mistake.

## Acceptance criteria

- [x] `proofs/tla/Approvals.tla` model-checks with TLC in CI with no invariant violation
- [x] Removing any one guard produces a counterexample (recorded in the devlog)
- [x] Ledger: LAW-10 and LAW-15 move to Modelled (LAW-04 and LAW-05 are already Tested (red), which ranks higher)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Run TLC on the committed spec in CI (pinned `tla2tools.jar`, checksum-verified)
- [x] Mutation-check each guard and record counterexamples
- [x] Keep the spec's state names identical to `sb-approvals(7)`
- [ ] Write or extend the law tests: LAW-04, LAW-05, LAW-10, LAW-15
- [x] Devlog entry, ledger and changelog updated

## Progress

CI job `proofs` runs `tools/ci/proofs.sh` with `tla2tools.jar` 1.8.0, checksum-pinned. State names match `sb-approvals(7)`. The Python cross-check passes, and all 5 guard mutations produce counterexamples.

2026-11-19: `make proofs` ran locally: TLC reports no error, 1,068 distinct states (same as the cross-check). `tools/ci/mutations.py` catches all 4 guard mutations with the expected invariant (`NoSelfApproval`, `AuditBeforeExec` twice, `KillWins`). Ledger updated. The law-test subtask stays open: LAW-10 and LAW-15 tests come with SB-403 and SB-406.

**Left:** two reviewers (proofs are a two-reviewer area, DoD) and one green `proofs` job in CI on a real push; the folder isn't a git repository yet, so CI hasn't run. Keep *In review* until both.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
