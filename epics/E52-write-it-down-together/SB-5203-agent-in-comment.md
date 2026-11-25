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
| Status | Open |

## Why

A reviewer who wants a department's view has to leave the draft to ask.

## Acceptance criteria

- [ ] `@NAME` in a section comment asks that department agent about the section, as the commenter
- [ ] The reply appears under the comment, cited only with what the commenter can open
- [ ] A reply is never a source and never part of the draft until someone accepts it; consultation stays one level deep

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Mention parsing
- [ ] Reply as the commenter
- [ ] Tests
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
