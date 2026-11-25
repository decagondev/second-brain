---
title: "E48 Ways of Working"
summary: "The company's recurring moments (a new joiner, a project, a quarter, a reorg, a 1:1) start the right drafts at the right time, without a steward remembering each step."
---

# E48: Ways of Working

> The company's recurring moments (a new joiner, a project, a quarter, a reorg, a 1:1) start the right drafts at the right time, without a steward remembering each step.

**Waves:** W47 · **Tickets:** 7 · **Stories:** 2

Planned 2026-11-25 after W46, with the user. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, the AI label never comes off by itself, a rhythm only drafts and stops with its subject, and a brief about a person draws only on what they published.

## User stories

- **US-48.1**: As a *steward*, *project lead*, *new joiner* or *manager*, I want the company's recurring moments to start the right drafts at the right time, so that no step depends on someone remembering it. Delivered by SB-4801, SB-4802, SB-4803, SB-4804.
- **US-48.2**: As a *leader*, *admin* or *manager*, I want planning, reorgs and 1:1s prepared from what is published, so that we decide with the facts and nothing moves unseen. Delivered by SB-4805, SB-4806, SB-4807.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S47.1 | Rhythms | Rhythms, a new joiner's first 90 days, project kickoff, project close-out | SB-4801, SB-4802, SB-4803, SB-4804 |
| S47.2 | Plan and reorganise | Quarterly planning, reorg dry run, 1:1 prep | SB-4805, SB-4806, SB-4807 |

## Exit gate

- [ ] No rhythm step publishes or sends without its named approver, and a rhythm stops when its subject leaves (LAW-29 green)
- [ ] A reorg plan matches the tree after it is applied, and a move that would widen access is refused
- [ ] A 1:1 brief draws only on published work, and the report sees the same brief (LAW-30 green)
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-4801](SB-4801-rhythms.md) | W47 / S47.1 | Rhythms | LAW-16, LAW-26, LAW-29 |
| [SB-4802](SB-4802-first-90-days.md) | W47 / S47.1 | A new joiner's first 90 days | LAW-01, LAW-12, LAW-29 |
| [SB-4803](SB-4803-project-kickoff.md) | W47 / S47.1 | Project kickoff | LAW-01, LAW-24, LAW-29 |
| [SB-4804](SB-4804-project-close-out.md) | W47 / S47.1 | Project close-out | LAW-01, LAW-14, LAW-16 |
| [SB-4805](SB-4805-quarterly-planning.md) | W47 / S47.2 | Quarterly planning | LAW-01, LAW-16, LAW-24 |
| [SB-4806](SB-4806-reorg-dry-run.md) | W47 / S47.2 | Reorg dry run | LAW-11, LAW-24 |
| [SB-4807](SB-4807-one-on-one-prep.md) | W47 / S47.2 | 1:1 prep | LAW-01, LAW-12, LAW-30 |

## Decisions for the user

Approved by the user on 2026-11-25 ([20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md#decisions-for-the-user)), every recommended decision: rhythms built in and steward-proposed, every step a draft a named person approves (SB-4801, LAW-29); 1:1 prep from published work only, with the report seeing the same brief (SB-4807, LAW-30).
