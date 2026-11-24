---
title: "SB-4307: Numbers from tables"
summary: "A model asked to add up a sheet gets the arithmetic wrong, and the answer looks just as sure."
---

# SB-4307: Numbers from tables

| Field | Value |
|---|---|
| Epic | [E43 Work That Comes to You](README.md) |
| Wave / slice | W42 / S42.2 |
| Stories | US-43.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | SB-2206 |
| Status | Open |

## Why

A model asked to add up a sheet gets the arithmetic wrong, and the answer looks just as sure.

## Acceptance criteria

- [ ] The `ask` agent may call one fixed read-only `table` tool: filter, then count, sum, average, minimum or maximum
- [ ] The tool reads stored chunks as the asker and returns the cell range used, which the answer cites
- [ ] A computed number not returned by the tool is dropped (LAW-14)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Tool
- [ ] Composer check
- [ ] Eval suite
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
