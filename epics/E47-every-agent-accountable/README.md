---
title: "E47 Every Agent Accountable"
summary: "Every agent's use is measured, any steward can stop it where it speaks, its cost is capped, its changes are tried before they ship, and its record can be handed to an auditor."
---

# E47: Every Agent Accountable

> Every agent's use is measured, any steward can stop it where it speaks, its cost is capped, its changes are tried before they ship, and its record can be handed to an auditor.

**Waves:** W46 · **Tickets:** 7 · **Stories:** 2

Planned 2026-11-25 after W43, with the user. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, and the AI label never comes off by itself.

## User stories

- **US-47.1**: As an *agent owner*, *steward* or *admin*, I want each agent's value, a pause wherever it speaks and a monthly cap, so that we keep the agents that earn their place. Delivered by SB-4701, SB-4702, SB-4703.
- **US-47.2**: As an *agent owner*, *admin* or *auditor*, I want shadow runs, agent access reviews, an AI mark on everything agents write and one record per agent, so that we can prove how every agent behaves. Delivered by SB-4704, SB-4705, SB-4706, SB-4707.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S46.1 | Value and control | Agent value, pause from the message, agent spend caps | SB-4701, SB-4702, SB-4703 |
| S46.2 | Prove it | Shadow runs, agent access review, marked as AI, an agent's record | SB-4704, SB-4705, SB-4706, SB-4707 |

## Exit gate

- [x] Every agent's value counts match the audit log
- [x] A paused agent runs nothing in that space, and a capped agent's scheduled runs stop
- [x] No shadow draft reaches anyone
- [x] Everything an agent wrote carries the AI mark (LAW-28 green)
- [x] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-4701](SB-4701-agent-value.md) | W46 / S46.1 | Agent value | LAW-07, LAW-11 |
| [SB-4702](SB-4702-pause-here.md) | W46 / S46.1 | Pause from the message | LAW-10, LAW-11 |
| [SB-4703](SB-4703-agent-spend-caps.md) | W46 / S46.1 | Agent spend caps | LAW-07 |
| [SB-4704](SB-4704-shadow-runs.md) | W46 / S46.2 | Shadow runs | LAW-01, LAW-06 |
| [SB-4705](SB-4705-agent-access-review.md) | W46 / S46.2 | Agent access review | LAW-11, LAW-16 |
| [SB-4706](SB-4706-marked-as-ai.md) | W46 / S46.2 | Marked as AI | LAW-28, LAW-11 |
| [SB-4707](SB-4707-agent-record.md) | W46 / S46.2 | An agent's record | LAW-07, LAW-11 |

## Decisions for the user

Approved by the user on 2026-11-25 ([19-departments-shape-account.md](../../docs/19-departments-shape-account.md#decisions-for-the-user)), every recommended decision: shadow runs for a week in the first ring, numbers for the owner and text only where they can open every cite (SB-4704); the AI mark always on and kept in published notes (SB-4706, LAW-28).
