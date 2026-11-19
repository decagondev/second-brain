---
title: "SB-1006: Ask in a space or company-wide"
summary: "People ask their department's brain most days and the whole company's sometimes."
---

# SB-1006: Ask in a space or company-wide

| Field | Value |
|---|---|
| Epic | [E10 Department Spaces](README.md) |
| Wave / slice | W10 / S10.3 Ask in a space |
| Stories | US-10.3 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md), `sb-spaces(7)` (new) |
| Laws | [LAW-01](../../manual/sb-laws.7.md#law-01), LAW-19 (new, W8), LAW-20 (new, W8) |
| Depends on | [SB-1005](SB-1005-ingest-routes-and-labels-every-document.md), [SB-302](../E04-retrieval-answers/SB-302-live-authorisation-of-every-chunk-before.md) |
| Status | Open |

## Why

People ask their department's brain most days and the whole company's sometimes.

## Acceptance criteria

- [ ] `POST /v1/ask` takes a space; company-wide is the union of the asker's spaces minus restricted content they don't hold
- [ ] Space, label and barrier checks run before the live ACL batch, and the ACL is still checked for every chunk
- [ ] LAW-20 green; the space-leak evals (SB-807) pass

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Retrieval filter
- [ ] Company-wide union
- [ ] Leak evals
- [ ] LAW-20 law test green
- [ ] Write or extend the law tests: LAW-01, LAW-19, LAW-20
- [ ] Devlog entry, ledger and changelog updated

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
