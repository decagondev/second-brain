---
title: "E35 Trust and Control"
summary: "People can see and remove what the brain keeps about them, and admins see access and spending coming."
---

# E35: Trust and Control

> People can see and remove what the brain keeps about them, and admins see access and spending coming.

**Waves:** W34 · **Tickets:** 6 · **Stories:** 2

Proposed 2026-11-24 after W31 and accepted by the user the same day, with every recommended decision. The concept is in [15-ask-capture-trust.md](../../docs/15-ask-capture-trust.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-35.1**: As a *person using the brain*, I want to download what the brain keeps about me, forget my questions, and see where I'm signed in, so that my data is mine to see and remove. Delivered by SB-3501, SB-3502, SB-3503.
- **US-35.2**: As a *steward* or *admin*, I want to review a restricted space's members, see the month's cost coming, and see usage by space, so that access and spending don't surprise us. Delivered by SB-3504, SB-3505, SB-3506.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S34.1 | Your own data | What the brain keeps about me, forget my questions, where I'm signed in | SB-3501, SB-3502, SB-3503 |
| S34.2 | See it before it goes wrong | Access review, month-end forecast, usage by space | SB-3504, SB-3505, SB-3506 |

## Exit gate

- [x] A person's export and erasure touch only their own records, and a legal hold still wins
- [x] Nobody sees another person's sessions
- [x] Access review changes no group; Remove asks the admin
- [x] No forecast from fewer than 3 days of data; no space row under 5 asks
- [x] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-3501](SB-3501-my-data.md) | W34 / S34.1 | What the brain keeps about me | LAW-01, LAW-11 |
| [SB-3502](SB-3502-forget-my-questions.md) | W34 / S34.1 | Forget my questions | LAW-07, LAW-11 |
| [SB-3503](SB-3503-my-sessions.md) | W34 / S34.1 | Where I'm signed in | LAW-11, LAW-12 |
| [SB-3504](SB-3504-access-review.md) | W34 / S34.2 | Access review | LAW-03, LAW-11, LAW-12 |
| [SB-3505](SB-3505-cost-forecast.md) | W34 / S34.2 | Month-end forecast | LAW-03 |
| [SB-3506](SB-3506-usage-by-space.md) | W34 / S34.2 | Usage by space | LAW-07, LAW-19 |

## Decisions for the user

Decided 2026-11-24 ([15-ask-capture-trust.md](../../docs/15-ask-capture-trust.md#decisions-for-the-user)): all of W32 to W34, in order; a disagreement mark only with two documents the asker can open (SB-3304); anyone can forget their own questions, and legal holds still win (SB-3502); old questions nudged at 5 working days and rerouted at 10 (SB-3404); access reviews every 90 days for restricted spaces, where Remove asks the admin (SB-3504).
