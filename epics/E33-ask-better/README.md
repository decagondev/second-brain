---
title: "E33 Ask Better"
summary: "An asker gets more out of each answer and knows what to ask."
---

# E33: Ask Better

> An asker gets more out of each answer and knows what to ask.

**Waves:** W32 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-24 after W31 and accepted by the user the same day, with every recommended decision. The concept is in [15-ask-capture-trust.md](../../docs/15-ask-capture-trust.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-33.1**: As an *asker*, I want to see the documents an answer left out, a term's meaning where it is used, a short answer, and where sources disagree, so that I get more out of each answer. Delivered by SB-3301, SB-3302, SB-3303, SB-3304.
- **US-33.2**: As an *asker*, I want to download a conversation, see an owner's other documents, and start from a space's suggested questions, so that I know where to look next. Delivered by SB-3305, SB-3306, SB-3307.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S32.1 | More from each answer | See also, glossary where it's used, short or full answer, sources disagree | SB-3301, SB-3302, SB-3303, SB-3304 |
| S32.2 | Know where to look | Download the conversation, their other documents, suggested questions | SB-3305, SB-3306, SB-3307 |

## Exit gate

- [ ] See also lists only documents the asker can open, never as claims
- [ ] A disagreement mark needs two documents the asker can open
- [ ] A short answer still has every claim cited or marked uncited
- [ ] Suggested questions reach only the space's members
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-3301](SB-3301-see-also.md) | W32 / S32.1 | See also | LAW-01, LAW-14 |
| [SB-3302](SB-3302-glossary-in-place.md) | W32 / S32.1 | Glossary where it's used | LAW-09, LAW-14 |
| [SB-3303](SB-3303-short-answer.md) | W32 / S32.1 | Short or full answer | LAW-14 |
| [SB-3304](SB-3304-sources-disagree.md) | W32 / S32.1 | Sources disagree | LAW-01, LAW-14 |
| [SB-3305](SB-3305-download-conversation.md) | W32 / S32.2 | Download the conversation | LAW-07, LAW-14 |
| [SB-3306](SB-3306-owner-documents.md) | W32 / S32.2 | Their other documents | LAW-01, LAW-12 |
| [SB-3307](SB-3307-suggested-questions.md) | W32 / S32.2 | Suggested questions | LAW-11, LAW-19 |

## Decisions for the user

Decided 2026-11-24 ([15-ask-capture-trust.md](../../docs/15-ask-capture-trust.md#decisions-for-the-user)): all of W32 to W34, in order; a disagreement mark only with two documents the asker can open (SB-3304); anyone can forget their own questions, and legal holds still win (SB-3502); old questions nudged at 5 working days and rerouted at 10 (SB-3404); access reviews every 90 days for restricted spaces, where Remove asks the admin (SB-3504).
