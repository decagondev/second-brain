---
title: "SB-010: Law harness and LAW-01 to LAW-05 as failing tests"
summary: "Laws written first fail for the right reason; then features turn them green."
---

# SB-010: Law harness and LAW-01 to LAW-05 as failing tests

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W0 / S0.4 Laws red, models green |
| Stories | US-00.3 |
| Manual | [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-03](../../manual/sb-laws.7.md#law-03), [LAW-04](../../manual/sb-laws.7.md#law-04), [LAW-05](../../manual/sb-laws.7.md#law-05) |
| Depends on | [SB-001](../E00-foundations/SB-001-repo-skeleton-and-toolchain.md) |
| Status | Done |

## Why

Laws written first fail for the right reason; then features turn them green.

## Acceptance criteria

- [x] `make laws` runs all law tests; each currently fails with `not implemented`, not with a crash
- [x] Each law test names its law ID in its test name and its failure message
- [x] Assurance ledger updated: these laws move from Stated to Tested (red)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Create `laws/` Go package with a `Law(id, fn)` helper
- [x] Write property tests for LAW-01 to LAW-05 over mock ports with `rapid`
- [x] Two seeded tenants fixture for LAW-02
- [x] Fault-injection fixture for LAW-03
- [x] Update ledger
- [x] Write or extend the law tests: LAW-01, LAW-02, LAW-03, LAW-04, LAW-05
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-18. `laws/`: `Check` helper (ADR-0016), world generator, checks for LAW-01 to LAW-05, and `expected_red.txt` with `tools/check/laws.py` as the ratchet. All five are red for the right reason.

Self-tests prove the checks work: the reference system passes, six mutations are caught, and the generator covers every hard case. The self-tests found a harness gap (BUG-0004): the generator needed public docs.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
