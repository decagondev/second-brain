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
| Status | Open |

## Why

Leaks between departments are an answer-quality failure as much as a policy one; the evals must try to cause them.

## Acceptance criteria

- [ ] A `space-leak` suite format: an asker, a space, a question that would need another space's restricted content, and the expected refusal
- [ ] Seed cases for People, Finance and Executive, run over the mock cell and failing until SB-1006

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Suite format in `sb-eval(5)`
- [ ] Seed cases
- [ ] Wire into `make evals` as expected-red
- [ ] Write or extend the law tests: LAW-20, LAW-22
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
