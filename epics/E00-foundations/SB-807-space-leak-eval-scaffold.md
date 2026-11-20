---
title: "SB-807: Space leak eval scaffold"
summary: "Leaks between departments are an answer-quality failure as much as a policy one; the evals must try to cause them."
---

# SB-807: Space leak eval scaffold

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W8 / S8.2 Laws red, model green |
| Stories | US-00.7 |
| Manual | [`sb-evals(7)`](../../manual/sb-evals.7.md), [`sb-eval(5)`](../../manual/sb-eval.5.md) |
| Laws | LAW-20 (new, W8), LAW-22 (new, W8) |
| Depends on | [SB-804](SB-804-law-19-to-law-23-as-failing-tests.md) |
| Status | Done |

## Why

Leaks between departments are an answer-quality failure as much as a policy one; the evals must try to cause them.

## Acceptance criteria

- [x] A `space-leak` suite format: an asker, a space, a question that would need another space's restricted content, and the expected refusal
- [x] Seed cases for People, Finance and Executive, run over the mock cell and failing until SB-1006

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Suite format in `sb-eval(5)`
- [x] Seed cases
- [x] Wire into `make evals` as expected-red
- [x] Write or extend the law tests: LAW-20, LAW-22
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-20 (slice S8.2): sb-eval(5) gained the `space` case field and "Space-leak suites", and sb-evals(7) lists `space-leak`. `evals/suites/space-leak.yaml` seeds People, Finance and Executive leak cases, plus two controls, over `evals/fixtures/space-leak/world.yaml`. Its world has restricted documents that a sharing mistake lets a non-member open at the source. On today's cell the three leak cases fail because the content really is cited, while the controls pass. `evals/expected_red.yaml` and `tools/check/evals_red.py` (in `make evals`, with a self-test) require exactly those cases to fail, so the suite is red for the right reason until SB-1006.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
