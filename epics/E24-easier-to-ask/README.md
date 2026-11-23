---
title: "E24 Easier to Ask"
summary: "An answer is easy to take away, and no screen leaves the asker without a next step."
---

# E24: Easier to Ask

> An answer is easy to take away, and no screen leaves the asker without a next step.

**Waves:** W23 · **Tickets:** 6 · **Stories:** 2

Proposed 2026-11-23 after W22 and accepted by the user the same day, with every recommended decision. The concept is in [12-ask-steward-operate.md](../../docs/12-ask-steward-operate.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-24.1**: As an *asker*, I want to copy, print and reach an answer from the keyboard, so that an answer goes where my work is. Delivered by SB-2401, SB-2402, SB-2403.
- **US-24.2**: As an *asker*, I want to narrow search, reuse my questions and always have a next step, so that I find things faster and never hit a dead end. Delivered by SB-2404, SB-2405, SB-2406.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S23.1 | Take the answer with you | Copy with cites, print, keyboard first | SB-2401, SB-2402, SB-2403 |
| S23.2 | Find it faster | Search filters, recent questions, no dead ends | SB-2404, SB-2405, SB-2406 |

## Exit gate

- [x] Copy and print keep every cite and the uncited mark
- [x] Ask, follow up, save and switch tabs with the keyboard alone; axe in each language
- [x] Filters only narrow; a hidden document never appears whatever the filters
- [x] Recent questions never leave the browser and are gone after sign-out
- [x] A hidden matching document changes no suggestion
- [x] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-2401](SB-2401-copy-answer.md) | W23 / S23.1 | Copy an answer with its cites | LAW-14, LAW-01 |
| [SB-2402](SB-2402-print-answer.md) | W23 / S23.1 | Print an answer | LAW-14 |
| [SB-2403](SB-2403-keyboard-first.md) | W23 / S23.1 | Keyboard first | LAW-14 |
| [SB-2404](SB-2404-search-filters.md) | W23 / S23.2 | Search filters | LAW-01 |
| [SB-2405](SB-2405-recent-questions.md) | W23 / S23.2 | Recent questions | LAW-07 |
| [SB-2406](SB-2406-no-dead-ends.md) | W23 / S23.2 | No dead ends | LAW-01 |

## Decisions for the user

Decided 2026-11-23 ([12-ask-steward-operate.md](../../docs/12-ask-steward-operate.md#decisions-for-the-user)): all of W23 to W25, in order; recent questions kept in the person's browser until sign-out (SB-2405); a soft one-hour claim that never blocks a reply (SB-2502); a pause of up to 30 days, with new questions still reaching the other stewards (SB-2506).
