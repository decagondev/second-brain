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
| Status | Done |

## Why

Asking a department for something happens in another tool, away from its knowledge.

## Acceptance criteria

- [x] A space can set `requests`, at most 10 categories, with `due_days`
- [x] Ask the department sends a request with a category to the steward queue, as a steward question with a due date
- [x] Claims, replies and escalation up the tree work unchanged; notifications carry counts only

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Manifest
- [x] Queue
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. Space requests { categories, due_days } (at most 10); GET and POST /v1/spaces/NAME/requests; StewardQuestion.Request {category, due_days, due} with ask.AddWorkingDays; the queue marks it overdue past due; notify climbAfter takes each step up the tree after due_days, told with counts only; web Requests (Ask the department) and the category and due date on the Questions tab. Tests: TestRequestsClimbWhenOverdue, TestRequestsAndBudgetsThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
