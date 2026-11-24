---
title: "SB-4004: Every agent in one place"
summary: "Nobody can see every agent, where it runs and how it's doing."
---

# SB-4004: Every agent in one place

| Field | Value |
|---|---|
| Epic | [E40 An Agent for Every Department](README.md) |
| Wave / slice | W39 / S39.2 |
| Stories | US-40.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-07, LAW-10 |
| Depends on | SB-4001 |
| Status | Done |

## Why

Nobody can see every agent, where it runs and how it's doing.

## Acceptance criteria

- [x] `GET /v1/admin/agents` lists each agent with its department, its spaces, runs and the share rated unhelpful in 30 days, its last eval result and kill state
- [x] Stewards see the same for their space's agents and propose a change as a manifest diff
- [x] Counts only; the list matches the manifest and the traces

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] Admin console
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

GET /v1/admin/agents and a steward's GET /v1/spaces/NAME/agents: department, spaces, 30-day runs and unhelpful ratings, eval and kill state; Propose these agents

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
