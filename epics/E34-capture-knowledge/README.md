---
title: "E34 Capture Knowledge"
summary: "A good answer, once given, is kept for the next person, and every document keeps someone looking after it."
---

# E34: Capture Knowledge

> A good answer, once given, is kept for the next person, and every document keeps someone looking after it.

**Waves:** W33 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-24 after W31 and accepted by the user the same day, with every recommended decision. The concept is in [15-ask-capture-trust.md](../../docs/15-ask-capture-trust.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-34.1**: As a *steward*, I want to publish a good reply as a note, spot repeated questions, have notes come due and old questions nudged, so that an answer given once is kept for the next person. Delivered by SB-3401, SB-3402, SB-3403, SB-3404.
- **US-34.2**: As a *steward* or *owner*, I want to name a caretaker for an owner-gone document, hear that my documents helped, and get glossary terms the documents define, so that every document keeps someone looking after it. Delivered by SB-3405, SB-3406, SB-3407.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S33.1 | Keep the good replies | Reply to note, spot a repeat, notes come due, nudge old questions | SB-3401, SB-3402, SB-3403, SB-3404 |
| S33.2 | Keep someone on every document | Take over a document, your documents helped, terms the documents define | SB-3405, SB-3406, SB-3407 |

## Exit gate

- [x] A published reply names no asker
- [x] No asker sees another asker's question
- [x] A deprovisioned caretaker is dropped
- [x] A suggested term is only a proposal
- [x] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-3401](SB-3401-reply-to-note.md) | W33 / S33.1 | Reply to note | LAW-07, LAW-11, LAW-16 |
| [SB-3402](SB-3402-spot-repeat.md) | W33 / S33.1 | Spot a repeat | LAW-01, LAW-07 |
| [SB-3403](SB-3403-notes-due.md) | W33 / S33.1 | Notes come due | LAW-13 |
| [SB-3404](SB-3404-nudge-old-questions.md) | W33 / S33.1 | Nudge old questions | LAW-07 |
| [SB-3405](SB-3405-caretaker.md) | W33 / S33.2 | Take over a document | LAW-11, LAW-12, LAW-19 |
| [SB-3406](SB-3406-documents-helped.md) | W33 / S33.2 | Your documents helped | LAW-07 |
| [SB-3407](SB-3407-defined-terms.md) | W33 / S33.2 | Terms the documents define | LAW-09, LAW-16 |

## Decisions for the user

Decided 2026-11-24 ([15-ask-capture-trust.md](../../docs/15-ask-capture-trust.md#decisions-for-the-user)): all of W32 to W34, in order; a disagreement mark only with two documents the asker can open (SB-3304); anyone can forget their own questions, and legal holds still win (SB-3502); old questions nudged at 5 working days and rerouted at 10 (SB-3404); access reviews every 90 days for restricted spaces, where Remove asks the admin (SB-3504).
