---
title: "SB-612: External penetration test and threat model refresh"
summary: "Independent eyes before scale."
---

# SB-612: External penetration test and threat model refresh

| Field | Value |
|---|---|
| Epic | [E07 Roll-out, Admin and Operations](README.md) |
| Wave / slice | W6 / S6.4 Harden |
| Stories | US-07.6 |
| Manual | [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | [LAW-02](../../manual/sb-laws.7.md#law-02), [LAW-07](../../manual/sb-laws.7.md#law-07), [LAW-09](../../manual/sb-laws.7.md#law-09) |
| Depends on | [SB-605](../E07-rollout-ops/SB-605-friendly-tenant-onboarding-unaided.md) |
| Status | In review |

## Why

Independent eyes before scale.

## Acceptance criteria

- [ ] No open critical or high findings (enforced by `tools/check/findings.py` in `make check`; true today for PT-0001 and PT-0002, and to be confirmed after the external test)
- [x] Threat model updated with findings

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [ ] Scope and schedule (scope and rules of engagement in `security/README.md`; booking the external team is human)
- [x] Fix findings
- [x] Update threat model and bugs log
- [x] Write or extend the law tests: LAW-02, LAW-07, LAW-09
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S6.4b): the manual changed first. sb-laws(7) gained "External review", and the LAW-02, LAW-07 and LAW-09 check lists name the new tests. sb-observability(7) says which secret encodings the redactor scrubs. `security/README.md` is the scope and rules of engagement, and `security/findings.yaml` is the register. `tools/check/findings.py`, which has its own self-test, fails `make check` while a critical or high finding isn't fixed, or when a fixed finding lacks its bug entry or isn't in the threat model. Three law tests were added before the external test. `TestLAW07_NoSecretInLogsEncoded` went red and found PT-0001 (BUG-0010, high, fixed). `TestLAW09_PenTestCorpusIsFlagged` went red over `security/attacks/injection.txt` and found PT-0002 (BUG-0011, medium, fixed). `TestLAW02_HostileHostsAreRefused` tries suffixes, userinfo, look-alike letters, percent-encoding, NUL and a second host, and was green first time. The threat model has a findings table and a new residual risk.

**Left:** the external pen test itself (human), and its findings.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
