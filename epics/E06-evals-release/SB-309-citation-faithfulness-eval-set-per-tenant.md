---
title: "SB-309: Citation-faithfulness eval set per tenant"
summary: "Faithfulness is measured, not asserted."
---

# SB-309: Citation-faithfulness eval set per tenant

| Field | Value |
|---|---|
| Epic | [E06 Evals, Release and Supply Chain](README.md) |
| Wave / slice | W3 / S3.3 Pilot |
| Stories | US-06.1 |
| Manual | [`sb-evals(7)`](../../manual/sb-evals.7.md), [`sb-eval(5)`](../../manual/sb-eval.5.md) |
| Laws | [LAW-14](../../manual/sb-laws.7.md#law-14) |
| Depends on | [SB-015](../E06-evals-release/SB-015-eval-harness-scaffold.md), [SB-304](../E04-retrieval-answers/SB-304-answer-composer-with-doc-version-citations.md) |
| Status | In review |

## Why

Faithfulness is measured, not asserted.

## Acceptance criteria

- [ ] Suite runs in CI and on `sb tenant verify` (CI: yes, in `make evals`; `sb tenant verify` is SB-601)
- [x] Each tenant can add golden questions in its repo folder

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Golden question format
- [x] Faithfulness scorer
- [x] Per-tenant suite loading
- [x] Write or extend the law tests: LAW-14
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S3.6): `evals/faithfulness.py` is the lexical scorer: the share of a claim's content words found in the cited chunk text, and 0 if the claim has a number the source doesn't. `faithful_min` uses it when a result carries no `support`, and each suite prints its citation score. `tools/evalcell` is a local cell: the real pipeline, index, live check (by email), cedar policies, ask agent and composer over a suite's `world.yaml`, with a deterministic `extractive` stand-in model and a `misattributing` one. `evals/suites/citation.yaml` runs in `make evals` (score 1.000, gate 0.92), every `tenants/<name>/evals/` suite runs as `tenant/<name>` (example: acme-logistics, gate 0.95, which must be stricter than the shared one), and `make evals` fails if the misattributing model passes. For LAW-14, the composer makes every claim cited or labelled in code; this eval measures whether the citations support the claims, and its self-test proves misattribution fails the gate. Building the local cell found BUG-0007.

**Left:** `sb tenant verify` (SB-601), and staging runs with the release's real model (SB-501), including a model judge if one is pinned.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
