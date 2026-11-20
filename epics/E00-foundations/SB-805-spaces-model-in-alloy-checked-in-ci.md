---
title: "SB-805: Spaces model in Alloy, checked in CI"
summary: "Narrowing, labels and barriers interact; a model finds the case nobody thought of before code does."
---

# SB-805: Spaces model in Alloy, checked in CI

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W8 / S8.2 Laws red, model green |
| Stories | US-00.7 |
| Manual | [`sb-permissions(7)`](../../manual/sb-permissions.7.md) |
| Laws | LAW-19 (new, W8), LAW-20 (new, W8), LAW-22 (new, W8) |
| Depends on | [SB-802](SB-802-permissions-and-policy-pages-for-spaces.md) |
| Status | Done |

## Why

Narrowing, labels and barriers interact; a model finds the case nobody thought of before code does.

## Acceptance criteria

- [x] `proofs/alloy/Spaces.als` extends `Permissions.als` with spaces, labels, barriers and company-wide asks
- [x] Assertions: no allowed read the source denies; no restricted read by a non-member; no read across a barrier; checked with no counterexample in CI
- [x] Mutations show each assertion can fail (a permit-past-source rule, a missing barrier)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Model
- [x] Assertions and scopes
- [x] Mutation checks in `tools/ci/mutations.py`
- [x] Python crosscheck mirror
- [x] Write or extend the law tests: LAW-19, LAW-20, LAW-22
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S8.2): `proofs/alloy/Spaces.als` models one company's cell with groups, spaces, labelled documents, barriers and the source's ACL, and the decision in sb-permissions(7) exactly. Four assertions (`SpacesOnlyNarrow`, `RestrictedStaysHome`, `BarriersHold`, `SpaceAskStaysInSpace`) are UNSAT at scope 5, and two runs are SAT, so the model isn't vacuous. `tools/ci/proofs.sh` picks it up in CI (`make proofs`). `tools/ci/mutations.py` adds three mutations: a permit past the source, a missing barrier, and restricted content in company-wide asks. Each makes exactly its law's check SAT, and the suite now catches 16 of 16. `proofs/crosscheck.py` mirrors the decision exhaustively over 6,912 small cases, with the same three mutations caught. The proofs README has the model and the mutations.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
