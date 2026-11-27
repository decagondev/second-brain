---
title: "SB-5406: How far each agent reaches"
summary: "Nobody can see how much an agent can read through its approvers, triggers and loans, or notice when that grows."
---

# SB-5406: How far each agent reaches

| Field | Value |
|---|---|
| Epic | [E54 Know Who Can See](README.md) |
| Wave / slice | W53 / S53.2 |
| Stories | US-54.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-audit(7)`](../../manual/sb-audit.7.md) |
| Laws | LAW-01, LAW-31 |
| Depends on | SB-4707, SB-5303 |
| Status | Done |

## Why

Nobody can see how much an agent can read through its approvers, triggers and loans, or notice when that grows.

## Acceptance criteria

- [x] `GET /v1/admin/agents/NAME/reach` and the registry show the spaces and number of documents an agent can read through its approvers, triggers, rhythms and open loans, with how many are labelled or held
- [x] Reach is in the agent's record (SB-4707) with the change since last week
- [x] Reach growing by more than a tenth in a week alerts the agent owner once

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Reach from a replay as each approver
- [x] Registry and record lines
- [x] Owner alert
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. `Cell.AgentReach` replays each space whose manifest enables the agent as each of its active stewards (`Agent.ReachIn`, every document checked live as them, at most 5000 a space) and counts documents, labelled ones and held ones; `GET /v1/admin/agents/NAME/reach` serves it and the agent's record carries it (`AgentRecord.Reach`). Each week's count is kept in `state/reach/AGENT.json`; more than a tenth more than last week (`ask.Grown`) tells the agent's owners once that week through the agent alerts. Open loans aren't counted: they last one run.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
