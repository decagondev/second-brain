---
title: "SB-2406: No dead ends"
summary: "No results, or a refusal, leaves the asker with nothing to do next."
---

# SB-2406: No dead ends

| Field | Value |
|---|---|
| Epic | [E24 Easier to Ask](README.md) |
| Wave / slice | W23 / S23.2 |
| Stories | US-24.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | SB-2404 |
| Status | Done |

## Why

No results, or a refusal, leaves the asker with nothing to do next.

## Acceptance criteria

- [x] No results or a refusal for lack of sources suggests: ask the whole company, clear filters, matching glossary terms of the space, ask the stewards
- [x] Suggestions come from the question, the filters and the space's glossary, never from documents
- [x] A test with a hidden matching document shows the same suggestions as with no document

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Suggestions
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-23 (S23.2). Documents with no results shows "You could try:" with **Search the whole company** (from a space), **Clear filters** (when a filter was set) and **Ask it as a question** (fills the ask box); an answer in a space with `no_sources` or `no_access` offers **Ask the whole company**. Which appear depends only on what the person typed and chose (`NextSteps` in `Documents.tsx`). Glossary terms stay where they were, under results and answers. Test: an empty search with three hidden documents shows exactly the same steps as one with none.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
