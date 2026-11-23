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
| Status | Done |

## Why

One long document can fill most of what the model reads.

## Acceptance criteria

- [x] At most 3 chunks of one document go into what the model reads for one answer, after the checks and the one-copy rule
- [x] Search results are unchanged
- [x] A long document matching every chunk gives 3; the eval suite doesn't regress

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Retrieval
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S28.1; tested in internal/core/ask/focus_test.go, internal/core/route/report_test.go, internal/cli/scopereport_test.go, tests/cell/retire_test.go and web/src/focus.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
