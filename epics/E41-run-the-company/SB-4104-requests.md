---
title: "SB-4104: Requests to a department"
summary: "Asking a department for something happens in another tool, away from its knowledge."
---

# SB-4104: Requests to a department

| Field | Value |
|---|---|
| Epic | [E41 Run the Company](README.md) |
| Wave / slice | W40 / S40.2 |
| Stories | US-41.2 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-3904 |
| Status | Open |

## Why

Asking a department for something happens in another tool, away from its knowledge.

## Acceptance criteria

- [ ] A space can set `requests`, at most 10 categories, with `due_days`
- [ ] Ask the department sends a request with a category to the steward queue, as a steward question with a due date
- [ ] Claims, replies and escalation up the tree work unchanged; notifications carry counts only

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Manifest
- [ ] Queue
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
