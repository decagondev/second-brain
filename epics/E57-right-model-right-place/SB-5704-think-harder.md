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
| Status | Done |

## Why

A hard research synthesis and a one-line lookup get the same model.

## Acceptance criteria

- [x] sb-cell(5) `model.deep` names a deeper model, or the main model with a higher reasoning effort, of the same provider
- [x] It is used only for research synthesis, tables, a consultation's synthesis and one retry after a failed citation check, only while gated, and never in a space whose routing policy forbids it
- [x] The trace records the tier and why; spend and limits count it at its own price

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Deep tier in the gateway
- [x] Gate it in the evals job
- [x] Routing policy
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. sb-cell(5) `model.deep` (`id`, `effort`, `price`) builds a third gateway of the main provider (`Gateway.Effort` sends `reasoning_effort` on OpenAI-compatible providers; Anthropic names a deeper id). `Agent.goDeeper` moves a research synthesis, a consultation and an answer with a table to it, and one retry of a mostly uncited answer is kept when it cites more (`DeepCitationRetry`); `Agent.deepAllowed` requires `DeepGated` (the evals job's `state/evals/deep-gated`, written when every case that passes on main passes on deep), a non-economy routing policy and a person under their limit. `Trace.Tier` is `deep` and `Trace.TierWhy` says why; spend and limits count it at `model.deep.price`, and the cost statement names its tier. Tested in TestThinkHarderOnlyWhenItHelps.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
