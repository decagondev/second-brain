---
title: "SB-4701: Agent value"
summary: "We know what an agent costs but not whether anyone uses what it drafts."
---

# SB-4701: Agent value

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.1 |
| Stories | US-47.1 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-4004, SB-4602 |
| Status | Done |

## Why

We know what an agent costs but not whether anyone uses what it drafts.

## Acceptance criteria

- [x] The registry and `GET /v1/admin/agents/value` show per agent and space, for 12 weeks, drafts made, published as is or edited, discarded, change rounds, time to publish, helpful rate and cost per published draft
- [x] Owners see their agents', admins all; a doubled discard rate tells the owner once
- [x] Counts only, matching the audit log

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Counts
- [x] API
- [x] UI
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. `ask.AgentValue` counts each agent's drafts per space over 12 weeks from the audit rows (`drafted`, `answered` with the reply's text compared to the draft's, `discarded`, `redrafted`), the thumbs on its answers and the `cost_usd` of its traces: counts only, never text. `GET /v1/admin/agents/value` (operators) and `GET /v1/agents/value` (an owner's own agents, by email or group) serve it; the web shows the table on the operators' Agents screen and, for an owner, on Questions, with Pause here on each row. A discard keeps its draft's agent, so a discarded draft is still counted and no longer shown as an agent run. `ask.DoubledDiscards` compares last ISO week's discard share with the week before's (at least 4 drafts) and the notify job tells the owner once a week (`agent` kind, keyed per agent and week). TestAgentValueCounts, TestDoubledDiscards, TestEveryAgentAccountableThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
