---
title: "SB-4703: Agent spend caps"
summary: "One busy agent can spend a month's budget before anyone sees it."
---

# SB-4703: Agent spend caps

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.1 |
| Stories | US-47.1 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-agents(7)`](../../manual/sb-agents.7.md) |
| Laws | LAW-07 |
| Depends on | SB-4105, SB-4701 |
| Status | Done |

## Why

One busy agent can spend a month's budget before anyone sees it.

## Acceptance criteria

- [x] sb-tenant(5) `agents: { NAME: { monthly_usd } }`
- [x] At 80% the owner is told; at 100% scheduled, triggered and shared runs pause until next month
- [x] On-demand asks keep working under each space's budget

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Manifest
- [x] Guard
- [x] Notify
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. sb-tenant(5) `agents` keeps its list form and gains a map form, `{ NAME: { monthly_usd: N } }` (1 to 100000); a legwork agent may be a key only to carry a cap. `ask.MonthSpend` sums an agent's trace `cost_usd` this UTC month in every space. The notify job tells the owner once at 80% and once at 100% (keyed per agent and month). From 100%, and whenever spend can't be read, drafts and routine runs stop with `EBUDGET`; on-demand asks go on under each space's budget. A manifest patch keeps the caps. TestAgentCaps, TestAPatchKeepsTheCaps, TestMonthSpend, TestEveryAgentAccountableThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
