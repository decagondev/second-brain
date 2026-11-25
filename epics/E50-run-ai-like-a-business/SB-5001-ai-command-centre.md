---
title: "SB-5001: AI command centre"
summary: "Spend is split across budgets, caps and value pages, and nobody sees AI use by feature and tier in one place."
---

# SB-5001: AI command centre

| Field | Value |
|---|---|
| Epic | [E50 Run AI Like a Business](README.md) |
| Wave / slice | W49 / S49.1 |
| Stories | US-50.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4701, SB-4703 |
| Status | Done |

## Why

Spend is split across budgets, caps and value pages, and nobody sees AI use by feature and tier in one place.

## Acceptance criteria

- [x] A costs page section and `GET /v1/admin/ai/usage` show 12 weeks of use and spend by space, feature, agent and tier, with budgets, caps and limits beside it
- [x] Totals match the traces, and a CSV download matches the page
- [x] People appear only as counts; any row under 5 shows as "fewer than 5"

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Usage query
- [x] Costs page section
- [x] CSV
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: every trace now carries its `feature` (ask, research, brief, routine, rhythm, consultation, eval), set by its caller. `internal/core/ask/aiusage.go` counts 12 weeks of traces by week, space, feature, agent and tier; `GET /v1/admin/ai/usage` (operators and approvers) adds each space's budget and spend, each capped agent's spend and the personal limit's bands, and `?format=csv` gives the same rows. A row under 5 answers shows `fewer_than_5` with its spend still in the totals; people are only ever counts. The admin console shows **AI use** with the CSV link. Prep and account briefs count as research, catch-up as a brief. Tested in TestAIUsageMasksSmallRows and TestSpendWithPurposeThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
