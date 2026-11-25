---
title: "SB-5204: Who made it"
summary: "A published note doesn't say which agent drafted it, who asked, who edited it and who approved it."
---

# SB-5204: Who made it

| Field | Value |
|---|---|
| Epic | [E52 Write It Down Together](README.md) |
| Wave / slice | W51 / S51.1 |
| Stories | US-52.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-citations(7)`](../../manual/sb-citations.7.md) |
| Laws | LAW-11, LAW-28 |
| Depends on | SB-4706, SB-5202 |
| Status | Done |

## Why

A published note doesn't say which agent drafted it, who asked, who edited it and who approved it.

## Acceptance criteria

- [x] A published note's front matter and history name the agent, the person who asked, each editor and the approver
- [x] The passage view and the note's page show them
- [x] History matches the audit log, and imported marks are kept

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Front matter
- [x] Passage view
- [x] Tests
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `markFront` adds `asked_by` (the person the draft ran as) and `edited_by` (who edited it in place) as addresses; `ingest.MarkOf` reads them, keeping only addresses (at most 10); `ports.Chunk.AskedBy/EditedBy`, Postgres columns `asked_by` and `edited_by`, `answer.Cite.asked_by/edited_by`, and `CiteMark` (chat apps and web) shows "asked for by …, edited by …". Tested in TestEditADraftTogether, TestCiteMarkNamesWhoMadeIt and TestMarkOfWhoMadeIt.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
