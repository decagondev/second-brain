---
title: "E41 Run the Company"
summary: "The work around knowledge (goals, updates, policies, requests and budgets) lives where the knowledge is, without becoming a second project tool."
---

# E41: Run the Company

> The work around knowledge (goals, updates, policies, requests and budgets) lives where the knowledge is, without becoming a second project tool.

**Waves:** W40 · **Tickets:** 6 · **Stories:** 2

Proposed 2026-11-24 after W37, waiting on the user's decisions. The concept is in [17-org-agents-os.md](../../docs/17-org-agents-os.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, and agents are definitions in the release bundle.

## User stories

- **US-41.1**: As a *leader* or *member*, I want goals, weekly updates and policies to read beside the knowledge they're about, so that I know what we're aiming at and what I must read. Delivered by SB-4101, SB-4102, SB-4103.
- **US-41.2**: As a *member*, *steward* or *admin*, I want to ask a department for something, keep agents within a budget, and have one home for each space, so that the space is where the department's work starts. Delivered by SB-4104, SB-4105, SB-4106.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S40.1 | Goals and updates | Goals, status updates, policies to read | SB-4101, SB-4102, SB-4103 |
| S40.2 | Requests, budgets and a home | Requests to a department, budgets, a home for each space | SB-4104, SB-4105, SB-4106 |

## Exit gate

- [ ] Goals and policies stay in the source; the brain only reads them
- [ ] Nothing an agent drafts is published without a steward
- [ ] A space over budget pauses scheduled runs and still answers asks
- [ ] Every part of the home is checked as the reader and hidden when empty
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-4101](SB-4101-goals.md) | W40 / S40.1 | Goals | LAW-01, LAW-13 |
| [SB-4102](SB-4102-status-updates.md) | W40 / S40.1 | Status updates | LAW-01, LAW-16 |
| [SB-4103](SB-4103-policies-to-read.md) | W40 / S40.1 | Policies to read | LAW-01, LAW-07, LAW-11 |
| [SB-4104](SB-4104-requests.md) | W40 / S40.2 | Requests to a department | LAW-07, LAW-11 |
| [SB-4105](SB-4105-department-budgets.md) | W40 / S40.2 | Department budgets | LAW-07 |
| [SB-4106](SB-4106-space-home.md) | W40 / S40.2 | A home for each space | LAW-01 |

## Decisions for the user

Waiting on the user ([17-org-agents-os.md](../../docs/17-org-agents-os.md#decisions-for-the-user)). Recommended: all of W38 to W40, in order; a tree of at most three levels where a child is only stricter and members aren't inherited (SB-3901); agents stay definitions in the release bundle (SB-4001 to SB-4003); one weekly brief a space and two fixed triggers, drafts only and no tools (SB-4005, SB-4006); stewards see by name who hasn't read a policy (SB-4103); budgets warn at 80% and pause scheduled runs at 100% (SB-4105).
