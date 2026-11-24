---
title: "E37 Capture Conversations"
summary: "What people decide in meetings and chat reaches the brain, after a steward's click."
---

# E37: Capture Conversations

> What people decide in meetings and chat reaches the brain, after a steward's click.

**Waves:** W36 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-24 after W34 and accepted by the user the same day, with every recommended decision. The concept is in [16-trust-capture-reach.md](../../docs/16-trust-capture-reach.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-37.1**: As a *person in meetings and chat*, I want transcripts and decisions to be found and cited, and to save a chat thread to the brain, so that what we decide isn't lost. Delivered by SB-3701, SB-3702, SB-3703.
- **US-37.2**: As a *steward* or *asker*, I want to see question trends, draft the missing document, ask a document's owner, and see what's already written before publishing, so that we fill each gap once. Delivered by SB-3704, SB-3705, SB-3706, SB-3707.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S36.1 | From meetings and chat | Meeting transcripts, decisions, save a chat thread, question trends | SB-3701, SB-3702, SB-3703, SB-3704 |
| S36.2 | Fill it in once | Write the missing document, ask the owner, already written? | SB-3705, SB-3706, SB-3707 |

## Exit gate

- [ ] Nothing from chat reaches the index without a steward's click
- [ ] The brain records no meeting; it reads only transcripts in the share
- [ ] Trends show only terms with at least 5 asks
- [ ] A deprovisioned owner can't be asked
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-3701](SB-3701-meeting-transcripts.md) | W36 / S36.1 | Meeting transcripts | LAW-01, LAW-09 |
| [SB-3702](SB-3702-decisions.md) | W36 / S36.1 | Decisions | LAW-01 |
| [SB-3703](SB-3703-save-chat-thread.md) | W36 / S36.1 | Save a chat thread | LAW-11, LAW-12, LAW-16 |
| [SB-3704](SB-3704-question-trends.md) | W36 / S36.1 | Question trends | LAW-07 |
| [SB-3705](SB-3705-draft-missing.md) | W36 / S36.2 | Write the missing document | LAW-16 |
| [SB-3706](SB-3706-ask-the-owner.md) | W36 / S36.2 | Ask the owner | LAW-07, LAW-11, LAW-12 |
| [SB-3707](SB-3707-already-written.md) | W36 / S36.2 | Already written? | LAW-01 |

## Decisions for the user

Decided 2026-11-24 ([16-trust-capture-reach.md](../../docs/16-trust-capture-reach.md#decisions-for-the-user)): all of W35 to W37, in order; a pinned answer shows first with every cite checked again as the asker (SB-3601); MCP read-only with the person's own token, off until the admin turns it on (SB-3801); one file of up to 2 MB per ask, never kept (SB-3802); a report is sent by the asker's choice and names them (SB-3606).
