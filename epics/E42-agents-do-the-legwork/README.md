---
title: "E42 Agents That Do the Legwork"
summary: "Agents prepare, research and follow up for people, and every result is still a cited draft or a message to the person themselves."
---

# E42: Agents That Do the Legwork

> Agents prepare, research and follow up for people, and every result is still a cited draft or a message to the person themselves.

**Waves:** W41 · **Tickets:** 7 · **Stories:** 2

Planned 2026-11-24 after W40, with the user. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, and every long run has a ceiling.

## User stories

- **US-42.1**: As an *asker*, I want research reports and meeting prep from what I can open, so that I start from the evidence rather than a blank page. Delivered by SB-4201, SB-4202, SB-4203.
- **US-42.2**: As a *steward*, *owner* or *goal owner*, I want incident briefs, customer themes, fix drafts and goal check-ins drafted for me, so that I keep things current without doing the digging. Delivered by SB-4204, SB-4205, SB-4206, SB-4207.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S41.1 | Research and prepare | Research reports, meeting prep, action items from meetings | SB-4201, SB-4202, SB-4203 |
| S41.2 | Follow up | Incident brief, voice of the customer, draft the fix, goal check-ins | SB-4204, SB-4205, SB-4206, SB-4207 |

## Exit gate

- [ ] A research plan runs only when the person presses Run, within its cost cap
- [ ] Every new agent has an eval suite that passes the gate and calls no tool
- [ ] Nothing an agent drafts reaches anyone before a steward, owner or the person acts on it
- [ ] Every cite in a stored result is checked again as the reader (LAW-26 green)
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-4201](SB-4201-research-reports.md) | W41 / S41.1 | Research reports | LAW-01, LAW-14, LAW-26 |
| [SB-4202](SB-4202-meeting-prep.md) | W41 / S41.1 | Meeting prep | LAW-01, LAW-09, LAW-12 |
| [SB-4203](SB-4203-action-items.md) | W41 / S41.1 | Action items from meetings | LAW-01, LAW-16 |
| [SB-4204](SB-4204-incident-brief.md) | W41 / S41.2 | Incident brief | LAW-01, LAW-09, LAW-16 |
| [SB-4205](SB-4205-voice-of-customer.md) | W41 / S41.2 | Voice of the customer | LAW-01, LAW-07, LAW-14 |
| [SB-4206](SB-4206-draft-the-fix.md) | W41 / S41.2 | Draft the fix | LAW-01, LAW-14, LAW-16 |
| [SB-4207](SB-4207-goal-check-ins.md) | W41 / S41.2 | Goal check-ins | LAW-01, LAW-16, LAW-24 |

## Decisions for the user

Approved by the user on 2026-11-24 ([18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md#decisions-for-the-user)), every recommended decision: all of W41 to W43, in order; the plan is shown and runs only on Run, at most 10 sub-questions and $2 a report (SB-4201); a pasted agenda plus a read-only Google Calendar connector (SB-4202); a steward confirms action items before owners are told (SB-4203).
