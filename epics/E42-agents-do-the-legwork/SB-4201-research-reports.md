---
title: "SB-4201: Research reports"
summary: "A question that needs many documents gets one shallow answer, and nobody has time to ask the ten follow-ups."
---

# SB-4201: Research reports

| Field | Value |
|---|---|
| Epic | [E42 Agents That Do the Legwork](README.md) |
| Wave / slice | W41 / S41.1 |
| Stories | US-42.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-01, LAW-14, LAW-26 |
| Depends on | SB-4105 |
| Status | Done |

## Why

A question that needs many documents gets one shallow answer, and nobody has time to ask the ten follow-ups.

## Acceptance criteria

- [x] **Research** shows a plan of at most 10 sub-questions and runs only when the person presses **Run**
- [x] Each sub-question is asked on the `ask` path as the person; the report cites every claim, and an uncited claim is dropped
- [x] A run stops at `research_usd` or the person's space budget; the report is kept 30 days and every cite is checked again as the reader on opening (LAW-26)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Agent and prompt
- [x] Job
- [x] API
- [x] UI
- [x] Eval suite
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. ask.Legwork Plan/Run/List/Open, the research agent and prompt (the model sees the question only), ResearchStore in memory and Postgres, budget.Tally from the gateway for research_usd (sb-tenant(5)), the notify job's Research part runs at most 3 queued reports a run as their owner. Agent.Recheck checks every stored cite as the reader on opening (LAW-26). API /v1/research; web Legwork tab. Tests: TestPlanSteps, TestTallyCountsOneContextsCalls, TestLegworkThroughARunningCell, agent-research suite, TestLAW26_AStoredResultIsCheckedWhenRead and TestSelf_LAW26CatchesAStaleReport.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
