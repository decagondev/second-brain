---
title: "SB-501: Eval runner and suites"
summary: "Permission, citation, injection and refusal suites decide releases."
---

# SB-501: Eval runner and suites

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W5 / S5.1 Eval gate |
| Stories | US-06.1 |
| Manual | [`sb-eval(1)`](../../manual/sb-eval.1.md), [`sb-eval(5)`](../../manual/sb-eval.5.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), [LAW-09](../../manual/sb-laws.7.md#law-09), [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-015](../E06-evals-release/SB-015-eval-harness-scaffold.md), [SB-309](../E06-evals-release/SB-309-citation-faithfulness-eval-set-per-tenant.md) |
| Status | Open |

## Why

Permission, citation, injection and refusal suites decide releases.

## Acceptance criteria

- [ ] Suites: permission, citation, injection, refusal, regression
- [ ] Deterministic seeds; results stored per release id

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Suite files
- [ ] Runner against staging cell
- [ ] Result store
- [ ] Write or extend the law tests: LAW-01, LAW-09, LAW-14
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
