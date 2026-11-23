---
title: "SB-2204: Wrong citations become eval cases"
summary: "A wrong-citation report is counted, not learned from."
---

# SB-2204: Wrong citations become eval cases

| Field | Value |
|---|---|
| Epic | [E22 Keep Knowledge Owned](README.md) |
| Wave / slice | W21 / S21.2 |
| Stories | US-22.2 |
| Manual | [`sb-evals(7)`](../../manual/sb-evals.7.md), [`sb-eval(5)`](../../manual/sb-eval.5.md) |
| Laws | LAW-06 |
| Depends on | — |
| Status | Done |

## Why

A wrong-citation report is counted, not learned from.

## Acceptance criteria

- [x] On the review list, a wrong-citation report gains Add to evals: the steward names what the answer should cite
- [x] The case (question, expected cite, space) joins the tenant's eval set; no answer is stored
- [x] A release that regresses the case isn't admitted to that tenant

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Case store
- [x] Gate reads tenant cases
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: built in S21.2. `reports` on review items, `/v1/spaces/NAME/evals`, `Agent.Probe`, the hourly `evals` job and `state/evals/hold.json`, read by the updater. Law test: `TestLAW06_AReleaseThatRegressesATenantCaseIsHeld`, self-test.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
