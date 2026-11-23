---
title: "E25 Easier to Steward"
summary: "A steward clears their list in minutes and doesn't repeat work, and nobody is pinged while they are away."
---

# E25: Easier to Steward

> A steward clears their list in minutes and doesn't repeat work, and nobody is pinged while they are away.

**Waves:** W24 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-23 after W22 and accepted by the user the same day, with every recommended decision. The concept is in [12-ask-steward-operate.md](../../docs/12-ask-steward-operate.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-25.1**: As a *steward*, I want to mark documents reviewed, claim questions and reuse replies, so that I clear the list without repeating work. Delivered by SB-2501, SB-2502, SB-2503.
- **US-25.2**: As a *steward*, I want to see what waits, take the list with me and pause while away, so that I know what needs me, and leave is quiet. Delivered by SB-2504, SB-2505, SB-2506, SB-2507.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S24.1 | Clear the list | Reviewed marks, claims, reply templates | SB-2501, SB-2502, SB-2503 |
| S24.2 | See what waits | Tab counts, CSV, pause, stewards' digest line | SB-2504, SB-2505, SB-2506, SB-2507 |

## Exit gate

- [ ] A reviewed document leaves the list until a new version, and a rebuild doesn't bring it back
- [ ] A removed steward's claims disappear at once
- [ ] The CSV holds exactly the rows on screen and no hidden title
- [ ] A paused steward gets nothing and the other stewards are still told
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-2501](SB-2501-reviewed-mark.md) | W24 / S24.1 | Reviewed, still right | LAW-13, LAW-11 |
| [SB-2502](SB-2502-claim-question.md) | W24 / S24.1 | Claim a question | LAW-21 |
| [SB-2503](SB-2503-reply-templates.md) | W24 / S24.1 | Reply templates | LAW-19 |
| [SB-2504](SB-2504-tab-counts.md) | W24 / S24.2 | Counts on the tabs | LAW-01 |
| [SB-2505](SB-2505-review-csv.md) | W24 / S24.2 | Review list as CSV | LAW-01 |
| [SB-2506](SB-2506-pause-notifications.md) | W24 / S24.2 | Pause notifications | — |
| [SB-2507](SB-2507-steward-digest.md) | W24 / S24.2 | Stewards' line in the digest | LAW-07, LAW-21 |

## Decisions for the user

Decided 2026-11-23 ([12-ask-steward-operate.md](../../docs/12-ask-steward-operate.md#decisions-for-the-user)): all of W23 to W25, in order; recent questions kept in the person's browser until sign-out (SB-2405); a soft one-hour claim that never blocks a reply (SB-2502); a pause of up to 30 days, with new questions still reaching the other stewards (SB-2506).
