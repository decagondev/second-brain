---
title: "E39 One Org, Many Levels"
summary: "Spaces follow the company's shape (company, department, team), and access only narrows down the tree."
---

# E39: One Org, Many Levels

> Spaces follow the company's shape (company, department, team), and access only narrows down the tree.

**Waves:** W38 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-24 after W37, waiting on the user's decisions. The concept is in [17-org-agents-os.md](../../docs/17-org-agents-os.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, and agents are definitions in the release bundle.

## User stories

- **US-39.1**: As an *admin* or *steward*, I want spaces to name a parent, settings to flow down, and asks to run across a department and its teams, so that the brain is shaped like the company. Delivered by SB-3901, SB-3902, SB-3903.
- **US-39.2**: As a *department leader* or *asker*, I want questions to climb the tree, counts to roll up, and to see who's who in a branch, so that nothing gets stuck in a team and leaders see their teams. Delivered by SB-3904, SB-3905, SB-3906, SB-3907.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S38.1 | The tree | Parent spaces, settings flow down, ask across a branch | SB-3901, SB-3902, SB-3903 |
| S38.2 | Leaders and escalation | Escalate up the tree, roll-ups, who's who, the tree never widens | SB-3904, SB-3905, SB-3906, SB-3907 |

## Exit gate

- [ ] The tree is at most three levels, with no cycles, and a child is never looser than its parent
- [ ] A branch ask returns only what each leaf space's own checks allowed
- [ ] Roll-ups carry counts only, with rows under 5 hidden
- [ ] LAW-24's law test is green, and red for the broken system
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-3901](SB-3901-parent-spaces.md) | W38 / S38.1 | Parent spaces | LAW-21, LAW-24 |
| [SB-3902](SB-3902-settings-flow-down.md) | W38 / S38.1 | Settings flow down | LAW-24 |
| [SB-3903](SB-3903-ask-across-branch.md) | W38 / S38.1 | Ask across a branch | LAW-01, LAW-24 |
| [SB-3904](SB-3904-escalate-up-tree.md) | W38 / S38.2 | Escalate up the tree | LAW-07, LAW-11 |
| [SB-3905](SB-3905-roll-ups.md) | W38 / S38.2 | Roll-ups for leaders | LAW-07, LAW-24 |
| [SB-3906](SB-3906-whos-who.md) | W38 / S38.2 | Who's who in a branch | LAW-01, LAW-12 |
| [SB-3907](SB-3907-tree-never-widens.md) | W38 / S38.2 | The tree never widens | LAW-24 |

## Decisions for the user

Waiting on the user ([17-org-agents-os.md](../../docs/17-org-agents-os.md#decisions-for-the-user)). Recommended: all of W38 to W40, in order; a tree of at most three levels where a child is only stricter and members aren't inherited (SB-3901); agents stay definitions in the release bundle (SB-4001 to SB-4003); one weekly brief a space and two fixed triggers, drafts only and no tools (SB-4005, SB-4006); stewards see by name who hasn't read a policy (SB-4103); budgets warn at 80% and pause scheduled runs at 100% (SB-4105).
