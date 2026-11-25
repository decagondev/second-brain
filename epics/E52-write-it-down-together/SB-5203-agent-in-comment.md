---
title: "SB-5203: Ask an agent in a comment"
summary: "A reviewer who wants a department's view has to leave the draft to ask."
---

# SB-5203: Ask an agent in a comment

| Field | Value |
|---|---|
| Epic | [E52 Write It Down Together](README.md) |
| Wave / slice | W51 / S51.1 |
| Stories | US-52.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-departments(7)`](../../manual/sb-departments.7.md) |
| Laws | LAW-01, LAW-09, LAW-27 |
| Depends on | SB-4902, SB-4501 |
| Status | Done |

## Why

A reviewer who wants a department's view has to leave the draft to ask.

## Acceptance criteria

- [x] `@NAME` in a section comment asks that department agent about the section, as the commenter
- [x] The reply appears under the comment, cited only with what the commenter can open
- [x] A reply is never a source and never part of the draft until someone accepts it; consultation stays one level deep

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Mention parsing
- [x] Reply as the commenter
- [x] Tests
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. A section comment starting `@NAME ` (an agent of the bundle) asks that agent the rest of the comment with up to 1,500 characters of the section, as the commenter in the draft's space (`Rescope`, the usual ask pipeline, no consultation), and keeps `reply: {agent, text, cites}` on the comment. Ask for changes reads only the comment's own words, never the reply. Tested in TestFixTheSourceThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
