---
title: "E46 Agents You Can Shape"
summary: "Stewards and owners shape what agents produce, without building agents in the browser, and anyone can see how an answer or draft was made."
---

# E46: Agents You Can Shape

> Stewards and owners shape what agents produce, without building agents in the browser, and anyone can see how an answer or draft was made.

**Waves:** W45 · **Tickets:** 6 · **Stories:** 2

Planned 2026-11-25 after W43, with the user. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, and the AI label never comes off by itself.

## User stories

- **US-46.1**: As a *steward* or *owner*, I want skills, a way to send a draft back with a note, and routines I can share with my space, so that agents write the way my department works. Delivered by SB-4601, SB-4602, SB-4603.
- **US-46.2**: As an *on-call engineer*, *account owner* or *asker*, I want post-incident reviews, account briefs and a view of how each answer was made, so that agents do more of the legwork and I can see what they did. Delivered by SB-4604, SB-4605, SB-4606.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S45.1 | Shape the work | Skills, ask for changes, shared routines | SB-4601, SB-4602, SB-4603 |
| S45.2 | Agents that go further | Post-incident review, account brief, how this was made | SB-4604, SB-4605, SB-4606 |

## Exit gate

- [ ] Every skill passes `sb agents lint` and its eval cases run in the gate
- [ ] A redraft cites only what its first draft could
- [ ] A shared routine runs as each member and its steward never sees their answers
- [ ] Every new agent has an eval suite that passes the gate and calls no tool
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-4601](SB-4601-skills.md) | W45 / S45.1 | Skills | LAW-04, LAW-06, LAW-09 |
| [SB-4602](SB-4602-ask-for-changes.md) | W45 / S45.1 | Ask for changes | LAW-09, LAW-14, LAW-16 |
| [SB-4603](SB-4603-shared-routines.md) | W45 / S45.1 | Shared routines | LAW-01, LAW-12, LAW-26 |
| [SB-4604](SB-4604-post-incident-review.md) | W45 / S45.2 | Post-incident review | LAW-01, LAW-09, LAW-16 |
| [SB-4605](SB-4605-account-brief.md) | W45 / S45.2 | Account brief | LAW-01, LAW-07, LAW-14 |
| [SB-4606](SB-4606-how-this-was-made.md) | W45 / S45.2 | How this was made | LAW-01, LAW-11, LAW-26 |

## Decisions for the user

Approved by the user on 2026-11-25 ([19-departments-shape-account.md](../../docs/19-departments-shape-account.md#decisions-for-the-user)), every recommended decision: skills are steward-proposed manifest entries with at least 3 eval cases, format and steps only (SB-4601); account briefs read-only from the CRM connector (SB-4605).
