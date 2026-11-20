---
title: "SB-1401: Fresh-eyes review of W8 to W12"
summary: "W8 to W12 went in fast; a review by someone who didn't write it finds what the author can't."
---

# SB-1401: Fresh-eyes review of W8 to W12

| Field | Value |
|---|---|
| Epic | [E14 Hardening](README.md) |
| Wave / slice | W13 / S13.1 |
| Stories | US-14.1 |
| Manual | as the change needs (manual first, P2) |
| Laws | LAW-19, LAW-20, LAW-22 |
| Depends on | [SB-1208](../E12-space-governance/SB-1208-cross-space-red-team-and-game-day.md) |
| Status | In progress |

## Why

W8 to W12 went in fast; a review by someone who didn't write it finds what the author can't.

## Acceptance criteria

- [ ] An independent review of every change from W8 to W12, the department brains, against the laws and the manual
- [ ] Every finding is fixed, or recorded with a reason in the findings register or the bug log
- [ ] Every fixed bug has a regression test that failed first

## Subtasks

- [ ] Update the man pages the change touches first (manual first, P2)
- [ ] Review
- [ ] Fix what it finds
- [ ] Regression tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

2026-11-20: three independent reviews (the authorisation path; governance and data lifecycle; connectors, ingest and the GCP runtime) reported 2 critical and about 30 high and medium findings. S13.1a fixed both critical ones and the misleading diff: PT-0005 (secrets split across chunks, missed forms; BUG-0014) and PT-0006/PT-0007 (steward values that read back differently, the diff hiding a change; BUG-0015), each with a regression test. S13.1b fixed the authorisation and approval findings: PT-0008 (BUG-0016) and PT-0009 (BUG-0017), each case with a regression test. S13.1c fixed the governance findings: PT-0010 (BUG-0018); PT-0011 (holds and reporting lines depend on the cell runtime's wiring) is accepted until the runtime exists. The rest follow in S13.1d.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
