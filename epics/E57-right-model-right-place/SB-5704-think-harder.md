---
title: "SB-5704: Think harder only when it helps"
summary: "A hard research synthesis and a one-line lookup get the same model."
---

# SB-5704: Think harder only when it helps

| Field | Value |
|---|---|
| Epic | [E57 Right Model, Right Place](README.md) |
| Wave / slice | W56 / S56.2 |
| Stories | US-57.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-evals(7)`](../../manual/sb-evals.7.md), [`sb-runtime(7)`](../../manual/sb-runtime.7.md) |
| Laws | LAW-06, LAW-14 |
| Depends on | SB-4405, SB-5002 |
| Status | Open |

## Why

A hard research synthesis and a one-line lookup get the same model.

## Acceptance criteria

- [ ] sb-cell(5) `model.deep` names a deeper model, or the main model with a higher reasoning effort, of the same provider
- [ ] It is used only for research synthesis, tables, a consultation's synthesis and one retry after a failed citation check, only while gated, and never in a space whose routing policy forbids it
- [ ] The trace records the tier and why; spend and limits count it at its own price

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Deep tier in the gateway
- [ ] Gate it in the evals job
- [ ] Routing policy
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
