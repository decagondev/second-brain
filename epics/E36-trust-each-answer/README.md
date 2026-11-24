---
title: "E36 Trust Each Answer"
summary: "An asker knows how far to trust an answer, and a steward can fix one that was wrong."
---

# E36: Trust Each Answer

> An asker knows how far to trust an answer, and a steward can fix one that was wrong.

**Waves:** W35 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-24 after W34 and accepted by the user the same day, with every recommended decision. The concept is in [16-trust-capture-reach.md](../../docs/16-trust-capture-reach.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-36.1**: As an *asker*, I want verified answers first, a verified-only mode, and documents that come due on their own interval or when people rate them unhelpful, so that I can rely on what I'm told. Delivered by SB-3601, SB-3602, SB-3603, SB-3604.
- **US-36.2**: As a *steward* or *asker*, I want to see conflicting documents, report a bad answer to the stewards, and see how many of an answer's sources are verified, so that a wrong answer gets fixed. Delivered by SB-3605, SB-3606, SB-3607.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S35.1 | Answers people can rely on | Verified answers, verified only, review every N days, unhelpful clears the mark | SB-3601, SB-3602, SB-3603, SB-3604 |
| S35.2 | When an answer goes wrong | Conflicts, report this answer, how sure the answer is | SB-3605, SB-3606, SB-3607 |

## Exit gate

- [ ] A pinned answer shows only with every cite checked again as the asker
- [ ] Verified only only narrows, and a mark at an old version never counts
- [ ] A report reaches the stewards only by the asker's click
- [ ] Conflicts name only documents the steward can open
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-3601](SB-3601-verified-answers.md) | W35 / S35.1 | Verified answers | LAW-01, LAW-14, LAW-19 |
| [SB-3602](SB-3602-verified-only.md) | W35 / S35.1 | Verified only | LAW-01, LAW-13 |
| [SB-3603](SB-3603-review-interval.md) | W35 / S35.1 | Review every N days | LAW-13 |
| [SB-3604](SB-3604-unhelpful-clears-mark.md) | W35 / S35.1 | Unhelpful clears the mark | LAW-07 |
| [SB-3605](SB-3605-conflicts.md) | W35 / S35.2 | Conflicts | LAW-01, LAW-07 |
| [SB-3606](SB-3606-report-answer.md) | W35 / S35.2 | Report this answer | LAW-01, LAW-07, LAW-11 |
| [SB-3607](SB-3607-verified-count.md) | W35 / S35.2 | How sure the answer is | LAW-14 |

## Decisions for the user

Decided 2026-11-24 ([16-trust-capture-reach.md](../../docs/16-trust-capture-reach.md#decisions-for-the-user)): all of W35 to W37, in order; a pinned answer shows first with every cite checked again as the asker (SB-3601); MCP read-only with the person's own token, off until the admin turns it on (SB-3801); one file of up to 2 MB per ask, never kept (SB-3802); a report is sent by the asker's choice and names them (SB-3606).
