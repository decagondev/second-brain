---
title: "E22 Keep Knowledge Owned"
summary: "Documents keep an owner after people move, the brain stops repeating the same mistake, and an asker who hits a wall can ask to get through it."
---

# E22: Keep Knowledge Owned

> Documents keep an owner after people move, the brain stops repeating the same mistake, and an asker who hits a wall can ask to get through it.

**Waves:** W21 · **Tickets:** 7 · **Stories:** 4

Proposed 2026-11-23 after W19 and accepted by the user the same day, with every recommended decision. The concept is in [11-loop-own-fit.md](../../docs/11-loop-own-fit.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, and no new page.

## User stories

- **US-22.1**: As a *steward*, I want to see documents with no owner, and copies of the same document, so that someone takes them over and answers cite one copy. Delivered by SB-2201, SB-2202.
- **US-22.2**: As a *steward*, I want a glossary for the space and wrong citations turned into eval cases, so that the brain understands our words and doesn't repeat a mistake. Delivered by SB-2203, SB-2204.
- **US-22.3**: As an *asker*, I want to ask the owners of sources I can't open for access, so that a hidden count isn't a dead end. Delivered by SB-2205.
- **US-22.4**: As an *asker* or *DPO*, I want spreadsheets read, and national ID numbers flagged, so that the brain reads what we keep and masks what it must. Delivered by SB-2206, SB-2207.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S21.1 | Owned | Orphaned documents, duplicates | SB-2201, SB-2202 |
| S21.2 | Right | Glossary, wrong citations as eval cases | SB-2203, SB-2204 |
| S21.3 | Through the wall | Ask for access | SB-2205 |
| S21.4 | Read more, flag more | Spreadsheets and CSV, national IDs | SB-2206, SB-2207 |

## Exit gate

- [x] A deprovisioned owner's documents reach the review list within a sync
- [x] A copied file is cited once
- [ ] "PO" finds "purchase order"; an instruction-like entry isn't used
- [ ] A release that regresses a tenant case is held
- [ ] An access request names the document only to its owner
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-2201](SB-2201-orphaned-documents.md) | W21 / S21.1 | Orphaned documents | LAW-12, LAW-01 |
| [SB-2202](SB-2202-duplicates.md) | W21 / S21.1 | Duplicate documents | LAW-13, LAW-01 |
| [SB-2203](SB-2203-glossary.md) | W21 / S21.2 | Space glossary | LAW-09, LAW-19 |
| [SB-2204](SB-2204-citation-evals.md) | W21 / S21.2 | Wrong citations become eval cases | LAW-06 |
| [SB-2205](SB-2205-ask-for-access.md) | W21 / S21.3 | Ask for access | LAW-01, LAW-07 |
| [SB-2206](SB-2206-spreadsheets.md) | W21 / S21.4 | Spreadsheets and CSV on file shares | LAW-09, LAW-13 |
| [SB-2207](SB-2207-national-ids.md) | W21 / S21.4 | National ID numbers as personal data | LAW-07 |

## Decisions for the user

Decided 2026-11-23 ([11-loop-own-fit.md](../../docs/11-loop-own-fit.md#decisions-for-the-user)): all of W20 to W22, in order; notifications by Slack or Mattermost direct message with email as the fallback (SB-2101, SB-2307); English, German and French (SB-2301); read-only API tokens, 90 days at most (SB-2306).
