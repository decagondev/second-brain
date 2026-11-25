---
title: "E50 Run AI Like a Business"
summary: "An admin sees what AI is used for and what it costs in one place, chooses how much model each task deserves, limits a person as well as an agent, gives agents targets, and learns whether a change actually helped."
---

# E50: Run AI Like a Business

> An admin sees what AI is used for and what it costs in one place, chooses how much model each task deserves, limits a person as well as an agent, gives agents targets, and learns whether a change actually helped.

**Waves:** W49 · **Tickets:** 7 · **Stories:** 2

Planned 2026-11-25 after W46, with the user. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, the AI label never comes off by itself, a rhythm only drafts and stops with its subject, and a brief about a person draws only on what they published.

## User stories

- **US-50.1**: As an *admin* or *agent owner*, I want AI use and spend in one place, routing policies, personal limits and agent targets, so that spend follows purpose. Delivered by SB-5001, SB-5002, SB-5003, SB-5004.
- **US-50.2**: As a *steward*, *agent owner*, *admin* or *auditor*, I want to know whether a change helped, a weekly digest and an inventory export, so that we learn what works and can show it. Delivered by SB-5005, SB-5006, SB-5007.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S49.1 | Spend with purpose | AI command centre, routing policies, personal limits, agent goals | SB-5001, SB-5002, SB-5003, SB-5004 |
| S49.2 | Learn what works | Did it help?, the admin's week, inventory export | SB-5005, SB-5006, SB-5007 |

## Exit gate

- [ ] The command centre's totals match the traces, and small rows are masked
- [ ] Each routing policy passes its space's eval suite in the gate
- [ ] At a personal limit asks still work on the economy tier, and review steps never stop
- [ ] Every check's numbers match the audit log and traces
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-5001](SB-5001-ai-command-centre.md) | W49 / S49.1 | AI command centre | LAW-07, LAW-11 |
| [SB-5002](SB-5002-routing-policies.md) | W49 / S49.1 | Routing policies | LAW-06, LAW-07 |
| [SB-5003](SB-5003-personal-limits.md) | W49 / S49.1 | Personal limits | LAW-07, LAW-11 |
| [SB-5004](SB-5004-agent-goals.md) | W49 / S49.1 | Agent goals | LAW-07, LAW-11 |
| [SB-5005](SB-5005-did-it-help.md) | W49 / S49.2 | Did it help? | LAW-07, LAW-11 |
| [SB-5006](SB-5006-admins-week.md) | W49 / S49.2 | The admin's week | LAW-07, LAW-11 |
| [SB-5007](SB-5007-inventory-export.md) | W49 / S49.2 | Inventory export | LAW-07, LAW-11, LAW-12 |

## Decisions for the user

Approved by the user on 2026-11-25 ([20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md#decisions-for-the-user)), every recommended decision: routing policies per space, `balanced` by default and capped by the parent (SB-5002); at a personal limit, drop to the economy tier rather than stop, and never stop review steps (SB-5003).
