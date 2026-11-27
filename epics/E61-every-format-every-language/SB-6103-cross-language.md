---
title: "SB-6103: Ask in one language, find another"
summary: "A question in German doesn't find the English policy that answers it."
---

# SB-6103: Ask in one language, find another

| Field | Value |
|---|---|
| Epic | [E61 Every Format, Every Language](README.md) |
| Wave / slice | W60 / S60.2 |
| Stories | US-61.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | LAW-06, LAW-14 |
| Depends on | SB-3003, SB-2706, SB-5501 |
| Status | Done |

## Why

A question in German doesn't find the English policy that answers it.

## Acceptance criteria

- [x] sb-cell(5) `languages` names the corpus's languages; a question is also searched in each, translated by the gated small tier with the glossary
- [x] Results are fused; cites stay in their own language and the answer is in the UI's
- [x] The translated queries are in the trace and the context view; with one language named, nothing is translated

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Query translation and fusion
- [x] Eval cases across languages
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. sb-cell(5) `languages`; with two or more, `Agent.translations` has the small tier (main while held) translate the question with the bundled `translate` prompt and the glossary's words; translations join `Query.Also` and the embedded text, an instruction-like one is dropped, and `Trace.Translated` shows in the asker's context view (`translated`). Tested in TestAskInOneLanguageFindAnother.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
