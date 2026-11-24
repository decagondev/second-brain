---
title: "E40 An Agent for Every Department"
summary: "Every department template has an agent worth asking, agents can run on a schedule or when something happens, and admins see every agent in one place."
---

# E40: An Agent for Every Department

> Every department template has an agent worth asking, agents can run on a schedule or when something happens, and admins see every agent in one place.

**Waves:** W39 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-24 after W37 and accepted by the user the same day, with every recommended decision. The concept is in [17-org-agents-os.md](../../docs/17-org-agents-os.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, and agents are definitions in the release bundle.

## User stories

- **US-40.1**: As a *person in any department*, I want an agent for my department that answers with cites, so that my department's work gets faster answers. Delivered by SB-4001, SB-4002, SB-4003.
- **US-40.2**: As an *admin* or *steward*, I want to see every agent, run one weekly, have a joiner and a transcript trigger one, and point askers to the right department, so that agents help without anyone remembering to ask. Delivered by SB-4004, SB-4005, SB-4006, SB-4007.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S39.1 | The catalogue | People, Legal, Sales, Support, Marketing, Operations, Design and Executive agents | SB-4001, SB-4002, SB-4003 |
| S39.2 | Agents that run themselves | Every agent in one place, on a schedule, when something happens, the right department | SB-4004, SB-4005, SB-4006, SB-4007 |

## Exit gate

- [x] Every new agent has an eval suite that passes the gate, and is refused outside its template's spaces
- [x] No new agent has a tool
- [x] A scheduled or triggered run is a draft until a steward publishes it, or a message to the person themselves
- [x] A space with no active steward runs nothing
- [x] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-4001](SB-4001-people-legal-agents.md) | W39 / S39.1 | People and Legal agents | LAW-01, LAW-09, LAW-14 |
| [SB-4002](SB-4002-sales-support-marketing-agents.md) | W39 / S39.1 | Sales, Support and Marketing agents | LAW-01, LAW-09, LAW-14 |
| [SB-4003](SB-4003-ops-design-exec-agents.md) | W39 / S39.1 | Operations, Design and Executive agents | LAW-01, LAW-09, LAW-14 |
| [SB-4004](SB-4004-agent-registry.md) | W39 / S39.2 | Every agent in one place | LAW-07, LAW-10 |
| [SB-4005](SB-4005-scheduled-agents.md) | W39 / S39.2 | Agents on a schedule | LAW-07, LAW-12, LAW-16 |
| [SB-4006](SB-4006-triggered-agents.md) | W39 / S39.2 | Agents when something happens | LAW-01, LAW-12, LAW-16 |
| [SB-4007](SB-4007-right-department.md) | W39 / S39.2 | The right department | LAW-01 |

## Decisions for the user

Decided 2026-11-24 ([17-org-agents-os.md](../../docs/17-org-agents-os.md#decisions-for-the-user)): all of W38 to W40, in order; a tree of at most three levels where a child is only stricter and members aren't inherited (SB-3901); agents stay definitions in the release bundle (SB-4001 to SB-4003); one weekly brief a space and two fixed triggers, drafts only and no tools (SB-4005, SB-4006); stewards see by name who hasn't read a policy (SB-4103); budgets warn at 80% and pause scheduled runs at 100% (SB-4105).
