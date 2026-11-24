---
title: "SB-3407: Terms the documents define"
summary: "Terms documents define themselves never reach the glossary."
---

# SB-3407: Terms the documents define

| Field | Value |
|---|---|
| Epic | [E34 Capture Knowledge](README.md) |
| Wave / slice | W33 / S33.2 |
| Stories | US-34.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-space(5)`](../../manual/sb-space.5.md) |
| Laws | LAW-09, LAW-16 |
| Depends on | SB-3107 |
| Status | Done |

## Why

Terms documents define themselves never reach the glossary.

## Acceptance criteria

- [x] The review list suggests terms defined like 'purchase order (PO)' in at least 2 of the space's documents and not in the glossary
- [x] Suggest fills the term proposal form; nothing changes until a proposal is applied
- [x] Instruction-shaped text is never suggested

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Scan
- [x] Review list
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-24. The review list's `terms` are definitions like `purchase order (PO)` or `PO (purchase order)`, initials checked, found in at least 2 of the space's documents the steward can open and in neither form in the glossary, at most 20. Chunks flagged `instruction_like` are never read, and a term the sanitiser would flag is skipped. **Suggest** only fills the Glossary form (SB-3107). Built on demand with the review list rather than in the weekly job: the same result, nothing kept. Tests: `TestDefinitions`, `TestCaretakersAndDefinedTermsThroughARunningCell`, web `caretaker.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
