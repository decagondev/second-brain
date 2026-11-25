---
title: "SB-5006: The admin's week"
summary: "An admin learns about new agents, hit caps and pauses one notification at a time."
---

# SB-5006: The admin's week

| Field | Value |
|---|---|
| Epic | [E50 Run AI Like a Business](README.md) |
| Wave / slice | W49 / S49.2 |
| Stories | US-50.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4702, SB-4406 |
| Status | Done |

## Why

An admin learns about new agents, hit caps and pauses one notification at a time.

## Acceptance criteria

- [x] Each Monday admins get a drafted digest of changed agents, skills and rhythms, caps and limits hit, pauses, guardrail trends, access reviews due and checks back
- [x] Every line links to its source record, and no document text appears
- [x] An admin can turn it off for themselves only

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Digest job
- [x] Links
- [x] Opt-out
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: `Cell.adminWeek`: from Monday 08:00 in the digest zone, each active member of `sb-operators` gets one `adminweek` notification for the week before: proposals approved with their approvers and manifests applied, agents at their cap and people at the limit, lifts, pauses and their reasons, doubled guardrail flags, open access reviews, checks back and the top 3 spaces by spend. Every line ends with its source record (an audit row, a review's evidence, a check or the page it counts from), and it carries counts and ids only. `PUT /v1/me/notifications` with `adminweek` off stops only that operator's copy. Rhythm changes appear as the proposals that changed them. Tested in TestAdminWeekText and TestLearnWhatWorksThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
