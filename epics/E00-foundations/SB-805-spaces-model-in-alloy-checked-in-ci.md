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
| Status | Open |

## Why

Narrowing, labels and barriers interact; a model finds the case nobody thought of before code does.

## Acceptance criteria

- [ ] `proofs/alloy/Spaces.als` extends `Permissions.als` with spaces, labels, barriers and company-wide asks
- [ ] Assertions: no allowed read the source denies; no restricted read by a non-member; no read across a barrier; checked with no counterexample in CI
- [ ] Mutations show each assertion can fail (a permit-past-source rule, a missing barrier)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Model
- [ ] Assertions and scopes
- [ ] Mutation checks in `tools/ci/mutations.py`
- [ ] Python crosscheck mirror
- [ ] Write or extend the law tests: LAW-19, LAW-20, LAW-22
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
