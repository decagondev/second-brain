---
title: "SB-3904: Escalate up the tree"
summary: "An unanswered team question stays stuck in the team."
---

# SB-3904: Escalate up the tree

| Field | Value |
|---|---|
| Epic | [E39 One Org, Many Levels](README.md) |
| Wave / slice | W38 / S38.2 |
| Stories | US-39.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-3404 |
| Status | Done |

## Why

An unanswered team question stays stuck in the team.

## Acceptance criteria

- [x] A question open 10 working days in a team goes to its department's stewards, and after 10 more to the company-wide answerers
- [x] It stays in each earlier queue, marked where it waited; each step is audited
- [x] A claim holds it; notifications carry counts only

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notify
- [x] Queue
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `notify.Job.Parent`, `Job.nextUp`: a team's question goes to its department's queue after 10 working days and on to company after 20, staying in each (`StewardQuestion.Up`), one audit row a step; `waited` tells each queue's stewards by count; `waitedFor` and `Stewards.queue` show it on the department's queue with `from_space`. TestQuestionsClimbTheTree, TestATeamQuestionOnTheDepartmentQueue.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
