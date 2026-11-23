---
title: "SB-2902: One document can't crowd out the rest"
summary: "One long document can fill most of what the model reads."
---

# SB-2902: One document can't crowd out the rest

| Field | Value |
|---|---|
| Epic | [E29 A Focused Context](README.md) |
| Wave / slice | W28 / S28.1 |
| Stories | US-29.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Open |

## Why

One long document can fill most of what the model reads.

## Acceptance criteria

- [ ] At most 3 chunks of one document go into what the model reads for one answer, after the checks and the one-copy rule
- [ ] Search results are unchanged
- [ ] A long document matching every chunk gives 3; the eval suite doesn't regress

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Retrieval
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
