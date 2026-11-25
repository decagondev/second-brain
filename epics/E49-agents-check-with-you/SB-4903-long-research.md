---
title: "SB-4903: Long research with checkpoints"
summary: "Hard questions need more than one pass, but a long run nobody can steer wastes money."
---

# SB-4903: Long research with checkpoints

| Field | Value |
|---|---|
| Epic | [E49 Agents That Check With You](README.md) |
| Wave / slice | W48 / S48.1 |
| Stories | US-49.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | LAW-01, LAW-26 |
| Depends on | SB-4201, SB-4105 |
| Status | Done |

## Why

Hard questions need more than one pass, but a long run nobody can steer wastes money.

## Acceptance criteria

- [x] `long: true` shows the plan and cost estimate first, and nothing runs before it is approved
- [x] At most 6 stages and 4 hours; at each checkpoint the asker can steer, stop or let it go on
- [x] A run stops at its ceiling (`research_usd`, default $5 for a long run) with what it has, and a document the asker lost mid-run is dropped at the next stage

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Plan approval
- [x] Stages and checkpoints
- [x] Cost ceiling
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `internal/core/ask/longresearch.go`: `long: true` plans with `long` (stages, an estimate at $0.10 a sub-question, and the ceiling: `research_usd`, or $5 when unset, so no new setting) and runs nothing until Run. The notify job runs one stage a run, at most 6 stages and 4 hours; before each stage the earlier sections are re-checked as the owner now (LAW-26). After each stage but the last the report waits at a checkpoint and its owner is notified; `POST /v1/research/{id}/steer`, `/stop` and `/go-on` steer, stop or continue it, and after 30 minutes it goes on by itself. It stops at its ceiling with what it has; a stage that hits the owner's asks-per-minute waits for the next run. Tested in TestLegworkThroughARunningCell. The web UI shows a long report like any report; steering is over the API.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
