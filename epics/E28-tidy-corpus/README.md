---
title: "E28 A Tidy Corpus"
summary: "Stewards and owners can see what has gone stale or unused, and take it out of answers without waiting on the source."
---

# E28: A Tidy Corpus

> Stewards and owners can see what has gone stale or unused, and take it out of answers without waiting on the source.

**Waves:** W27 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-23 after W25 and accepted by the user the same day, with every recommended decision. The concept is in [13-fresh-tidy-focused.md](../../docs/13-fresh-tidy-focused.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-28.1**: As a *steward*, I want to retire a superseded document from my space's answers and see what is duplicated, unused or rests on moved sources, so that answers come from what we keep now. Delivered by SB-2801, SB-2802, SB-2803, SB-2804.
- **US-28.2**: As a *steward* or *owner*, I want unused terms marked, a monthly word about my own old documents, and freshness in the report, so that upkeep is shared and visible. Delivered by SB-2805, SB-2806, SB-2807.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S27.1 | Take it out of answers | Retire from answers, same name different text, never cited, notes whose sources moved | SB-2801, SB-2802, SB-2803, SB-2804 |
| S27.2 | Keep upkeep shared | Unused glossary terms, owners' monthly note, freshness in the report | SB-2805, SB-2806, SB-2807 |

## Exit gate

- [ ] A retired document is never cited in its space, still is in another, and a rebuild keeps it out
- [ ] Every list counts what the steward can't open and names nothing in it
- [ ] An owner is told only of their own documents, once a month
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-2801](SB-2801-retire-from-answers.md) | W27 / S27.1 | Retire from answers | LAW-19, LAW-11, LAW-13 |
| [SB-2802](SB-2802-same-name-different-text.md) | W27 / S27.1 | Same name, different text | LAW-01 |
| [SB-2803](SB-2803-never-cited.md) | W27 / S27.1 | Never cited | LAW-01, LAW-07 |
| [SB-2804](SB-2804-notes-sources-moved.md) | W27 / S27.1 | Notes whose sources moved | LAW-01 |
| [SB-2805](SB-2805-unused-terms.md) | W27 / S27.2 | Unused glossary terms | LAW-07 |
| [SB-2806](SB-2806-owner-nudges.md) | W27 / S27.2 | Owners hear about their own old documents | LAW-01, LAW-07, LAW-12 |
| [SB-2807](SB-2807-freshness-report.md) | W27 / S27.2 | Freshness in the report | LAW-07 |

## Decisions for the user

Decided 2026-11-23 ([13-fresh-tidy-focused.md](../../docs/13-fresh-tidy-focused.md#decisions-for-the-user)): all of W26 to W28, in order; warn the asker when more than half of an answer's cites are stale (SB-2701); stewards retire a document from their own space's answers only, undoable and audited (SB-2801); owners' monthly note on by default, each person can turn it off (SB-2806); conversations send the last 8 questions (SB-2901).
