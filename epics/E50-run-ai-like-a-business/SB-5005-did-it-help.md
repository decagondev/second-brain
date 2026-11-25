---
title: "SB-5005: Did it help?"
summary: "Fixes, skills and agent changes ship, and nobody checks whether they moved anything."
---

# SB-5005: Did it help?

| Field | Value |
|---|---|
| Epic | [E50 Run AI Like a Business](README.md) |
| Wave / slice | W49 / S49.2 |
| Stories | US-50.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4701, SB-4507 |
| Status | Done |

## Why

Fixes, skills and agent changes ship, and nobody checks whether they moved anything.

## Acceptance criteria

- [x] A shipped fix, skill, routing policy, policy or agent version schedules a check 4 weeks later comparing the 4 weeks before and after
- [x] The numbers match the audit log and traces; fewer than 20 events either side says "not enough data"
- [x] A check drafts a short result for the steward or owner and never undoes anything

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Check scheduling
- [x] Before and after
- [x] Result draft
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: `internal/core/ask/checks.go`: the notify job finds the changes that shipped 4 weeks ago from the audit log (a steward proposal `approved`, a policy `policy_marked`, a fix whose document has a newer version) and the traces (an agent's first answer under a new release), and counts the 4 weeks either side in its space, or for its agent: answers, those that found nothing, helpful rate, discard rate and median hours to a steward's first reply. Fewer than 20 answers a side is "not enough data"; otherwise better, worse or no clear change. The result is kept in `state/checks`, listed at `GET /v1/admin/checks[/ID]` and `GET /v1/spaces/NAME/checks`, and drafted for the space's stewards or the agent's owner as one `check` notification. Nothing is undone. A proposal's own named numbers aren't stored yet, so every check uses the default numbers. Tested in TestChecksCountBeforeAndAfter and TestLearnWhatWorksThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
