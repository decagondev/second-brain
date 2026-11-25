---
title: "SB-4305: What agents did as me"
summary: "Scheduled briefs, triggers and routines run as people, who can't see what ran in their name."
---

# SB-4305: What agents did as me

| Field | Value |
|---|---|
| Epic | [E43 Work That Comes to You](README.md) |
| Wave / slice | W42 / S42.2 |
| Stories | US-43.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-audit(7)`](../../manual/sb-audit.7.md) |
| Laws | LAW-01, LAW-11 |
| Depends on | SB-4005 |
| Status | Done |

## Why

Scheduled briefs, triggers and routines run as people, who can't see what ran in their name.

## Acceptance criteria

- [x] `GET /v1/me/agent-runs` and **Agent runs** list every run made as me in the last 90 days, from traces and the audit log
- [x] Each shows when, which agent and space, what it read (titles I can open now, the rest counted) and what it drafted
- [x] No new record is kept

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. ask.Legwork.AgentRuns: agent drafts made as the person from the steward queues and their reports, briefs, routines and fixes, each run's traces' allowed chunks checked again with CheckCites as them now. GET /v1/me/agent-runs; no record of its own. Web: Agent runs on the Inbox tab. TestContextForTheWorkThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
