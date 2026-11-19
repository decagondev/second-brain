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
| Status | Open |

## Why

Faithfulness is measured, not asserted.

## Acceptance criteria

- [ ] Suite runs in CI and on `sb tenant verify`
- [ ] Each tenant can add golden questions in its repo folder

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Golden question format
- [ ] Faithfulness scorer
- [ ] Per-tenant suite loading
- [ ] Write or extend the law tests: LAW-14
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
