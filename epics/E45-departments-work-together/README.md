---
title: "E45 Departments Work Together"
summary: "A question or a piece of work that spans departments reaches the right ones, as the asker, and leaders see the week across their branch without assembling it."
---

# E45: Departments Work Together

> A question or a piece of work that spans departments reaches the right ones, as the asker, and leaders see the week across their branch without assembling it.

**Waves:** W44 · **Tickets:** 7 · **Stories:** 2

Planned 2026-11-25 after W43, with the user. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, and the AI label never comes off by itself.

## User stories

- **US-45.1**: As a *leader* or *asker*, I want department agents consulted as me, work handed to the right department and a drafted week for my branch, so that the company works as one brain, not a set of separate ones. Delivered by SB-4501, SB-4502, SB-4503.
- **US-45.2**: As a *steward* or *asker*, I want waits between departments, decisions that say what they replaced, catch-up and service levels, so that work between departments doesn't fall through the gaps. Delivered by SB-4504, SB-4505, SB-4506, SB-4507.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S44.1 | Agents across the tree | Ask each department, hand off to a department, the leader's week | SB-4501, SB-4502, SB-4503 |
| S44.2 | Work across departments | Waiting on another department, decisions that replace decisions, catch me up, department service levels | SB-4504, SB-4505, SB-4506, SB-4507 |

## Exit gate

- [x] A consultation runs as the asker, one level deep, and never reaches a space the asker isn't in (LAW-27 green)
- [x] Nothing an agent drafts for another department is sent without the asker
- [x] Every line of the leader's week and of a catch-up is cited or counted
- [x] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-4501](SB-4501-ask-each-department.md) | W44 / S44.1 | Ask each department | LAW-01, LAW-24, LAW-27 |
| [SB-4502](SB-4502-hand-off.md) | W44 / S44.1 | Hand off to a department | LAW-01, LAW-16 |
| [SB-4503](SB-4503-leaders-week.md) | W44 / S44.1 | The leader's week | LAW-01, LAW-16, LAW-24 |
| [SB-4504](SB-4504-waiting-on.md) | W44 / S44.2 | Waiting on another department | LAW-01, LAW-24 |
| [SB-4505](SB-4505-superseded-decisions.md) | W44 / S44.2 | Decisions that replace decisions | LAW-01, LAW-14 |
| [SB-4506](SB-4506-catch-me-up.md) | W44 / S44.2 | Catch me up | LAW-01, LAW-14, LAW-26 |
| [SB-4507](SB-4507-service-levels.md) | W44 / S44.2 | Department service levels | LAW-07, LAW-24 |

## Decisions for the user

Approved by the user on 2026-11-25 ([19-departments-shape-account.md](../../docs/19-departments-shape-account.md#decisions-for-the-user)), every recommended decision: all of W44 to W46, in order; consultation one level deep, at most 6 child spaces the asker belongs to, as the asker (SB-4501, LAW-27); the asker sends every handoff (SB-4502).
