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
| Status | Done |

## Why

A model asked to add up a sheet gets the arithmetic wrong, and the answer looks just as sure.

## Acceptance criteria

- [x] The `ask` agent may call one fixed read-only `table` tool: filter, then count, sum, average, minimum or maximum
- [x] The tool reads stored chunks as the asker and returns the cell range used, which the answer cites
- [x] A computed number not returned by the tool is dropped (LAW-14)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Tool
- [x] Composer check
- [x] Eval suite
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. ask/table.go: NumericQuestion, the table prompt (headers only, as data), readPlan, wholeText from every chunk of the version checked as the asker, runTable (count, sum, average, min, max, up to 3 filters), its result a passage with the sheet's cite and Response.Table; tableCheck drops a claim on a sheet whose number no passage or the tool holds. Extractive plans from the question's words; evals/suites/table.yaml (table_value check) in the gate. LAW-14: TestLAW14_NumbersFromTablesComeFromTheTool and TestSelf_LAW14CatchesAMadeUpNumber.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
