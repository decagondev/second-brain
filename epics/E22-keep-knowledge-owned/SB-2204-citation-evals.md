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
| Status | Open |

## Why

A wrong-citation report is counted, not learned from.

## Acceptance criteria

- [ ] On the review list, a wrong-citation report gains Add to evals: the steward names what the answer should cite
- [ ] The case (question, expected cite, space) joins the tenant's eval set; no answer is stored
- [ ] A release that regresses the case isn't admitted to that tenant

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Case store
- [ ] Gate reads tenant cases
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
