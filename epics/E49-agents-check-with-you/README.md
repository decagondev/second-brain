---
title: "E49 Agents That Check With You"
summary: "Agents ask before they guess, take comments where they belong, show a plan before a long run, and notice when their sources change."
---

# E49: Agents That Check With You

> Agents ask before they guess, take comments where they belong, show a plan before a long run, and notice when their sources change.

**Waves:** W48 · **Tickets:** 6 · **Stories:** 2

Planned 2026-11-25 after W46, with the user. The concept is in [20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, the AI label never comes off by itself, a rhythm only drafts and stops with its subject, and a brief about a person draws only on what they published.

## User stories

- **US-49.1**: As *anyone who asks or reviews*, I want agents to ask before guessing, take comments on a section and show a plan before long work, so that I steer the work instead of redoing it. Delivered by SB-4901, SB-4902, SB-4903.
- **US-49.2**: As a *steward* or *agent owner*, I want to test changes with scenarios, have drafts redrafted when their sources change, and offer good starter questions, so that agents keep up with the space. Delivered by SB-4904, SB-4905, SB-4906.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S48.1 | Check before acting | Ask before guessing, comments on a section, long research with checkpoints | SB-4901, SB-4902, SB-4903 |
| S48.2 | Agents that keep up | Scenarios before you approve, when a source changes, starter questions | SB-4904, SB-4905, SB-4906 |

## Exit gate

- [ ] An agent asks at most one question in a turn, and scheduled runs never ask
- [ ] A redraft changes only commented sections and cites only what the first draft could
- [ ] Long research runs nothing before its plan is approved and stops at its cost ceiling
- [ ] Nothing redrafted after a source change republishes without its approver
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-4901](SB-4901-ask-before-guessing.md) | W48 / S48.1 | Ask before guessing | LAW-09, LAW-26 |
| [SB-4902](SB-4902-section-comments.md) | W48 / S48.1 | Comments on a section | LAW-09, LAW-14, LAW-16 |
| [SB-4903](SB-4903-long-research.md) | W48 / S48.1 | Long research with checkpoints | LAW-01, LAW-26 |
| [SB-4904](SB-4904-scenarios.md) | W48 / S48.2 | Scenarios before you approve | LAW-01, LAW-06 |
| [SB-4905](SB-4905-source-changed.md) | W48 / S48.2 | When a source changes | LAW-01, LAW-14, LAW-26 |
| [SB-4906](SB-4906-starter-questions.md) | W48 / S48.2 | Starter questions | LAW-01, LAW-07 |

## Decisions for the user

Approved by the user on 2026-11-25 ([20-rhythms-check-run.md](../../docs/20-rhythms-check-run.md#decisions-for-the-user)), every recommended decision: at most one clarifying question with up to 4 choices, on-demand asks only (SB-4901); long research with plan approval first, up to 4 hours in 6 stages, $5 default ceiling (SB-4903).
