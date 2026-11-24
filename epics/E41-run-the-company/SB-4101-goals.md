---
title: "SB-4101: Goals"
summary: "Goals live in other tools, so answers can't cite them and leaders can't see them by space."
---

# SB-4101: Goals

| Field | Value |
|---|---|
| Epic | [E41 Run the Company](README.md) |
| Wave / slice | W40 / S40.1 |
| Stories | US-41.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-ingest(7)`](../../manual/sb-ingest.7.md) |
| Laws | LAW-01, LAW-13 |
| Depends on | SB-3702 |
| Status | Done |

## Why

Goals live in other tools, so answers can't cite them and leaders can't see them by space.

## Acceptance criteria

- [x] Front matter `type: goal` with `owner`, `quarter`, `status` and optional `parent_goal` types a document `goal`
- [x] `GET /v1/spaces/NAME/goals` lists the branch's goals by quarter, nested, each checked as the asker
- [x] A goal whose parent the asker can't open shows without it; goals stay in the source

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Ingest
- [x] API
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. ask.ParseGoal reads type: goal front matter; GET /v1/spaces/NAME/goals lists the branch's goals by quarter, nested by parent_goal, each checked with retrieve.Allowed as the caller, hidden counted; search type=goal; web Goals on the Start view. Tests: TestParseAndNestGoals, TestRunTheCompanyThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
