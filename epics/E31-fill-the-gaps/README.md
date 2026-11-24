---
title: "E31 Fill the Gaps"
summary: "Stewards see what people couldn't find, and the upkeep is spread across stewards and owners."
---

# E31: Fill the Gaps

> Stewards see what people couldn't find, and the upkeep is spread across stewards and owners.

**Waves:** W30 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-24 after W28 and accepted by the user the same day, with every recommended decision. The concept is in [14-evidence-gaps-surprises.md](../../docs/14-evidence-gaps-surprises.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-31.1**: As a *steward* or *admin*, I want to see the questions that found nothing, spaces with no steward, and the space's feedback, and to review in bulk, so that we write what's missing and nothing waits for nobody. Delivered by SB-3101, SB-3102, SB-3103, SB-3104.
- **US-31.2**: As a *steward* or *owner*, I want to see what falls due soon, mark my own documents reviewed, and suggest glossary terms from the screen, so that upkeep is shared before it's overdue. Delivered by SB-3105, SB-3106, SB-3107.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S30.1 | See what's missing | Questions that found nothing, spaces with no steward left, review several at once, feedback for the stewards | SB-3101, SB-3102, SB-3103, SB-3104 |
| S30.2 | Share the upkeep | Coming due, owners mark their own reviewed, suggest a glossary term | SB-3105, SB-3106, SB-3107 |

## Exit gate

- [ ] No question text appears in any count
- [ ] A space with no active steward warns and its questions reroute; a directory error warns
- [ ] Each document in a bulk action is checked and audited on its own
- [ ] An owner can mark only their own document
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-3101](SB-3101-found-nothing.md) | W30 / S30.1 | Questions that found nothing | LAW-07 |
| [SB-3102](SB-3102-no-steward-left.md) | W30 / S30.1 | Spaces with no steward left | LAW-12, LAW-03 |
| [SB-3103](SB-3103-bulk-review.md) | W30 / S30.1 | Review several at once | LAW-11, LAW-19 |
| [SB-3104](SB-3104-steward-feedback.md) | W30 / S30.1 | Feedback for the stewards | LAW-01, LAW-07 |
| [SB-3105](SB-3105-coming-due.md) | W30 / S30.2 | Coming due | LAW-01 |
| [SB-3106](SB-3106-owner-marks.md) | W30 / S30.2 | Owners mark their own reviewed | LAW-01, LAW-11, LAW-12 |
| [SB-3107](SB-3107-suggest-term.md) | W30 / S30.2 | Suggest a glossary term | LAW-16, LAW-11 |

## Decisions for the user

Decided 2026-11-24 ([14-evidence-gaps-surprises.md](../../docs/14-evidence-gaps-surprises.md#decisions-for-the-user)): all of W29 to W31, in order; answers in the language picked in the UI, or the manifest's in chat (SB-3003); passages of up to 600 characters (SB-3001); owners mark only their own documents reviewed (SB-3106); admins alone set the notice, ending within 7 days (SB-3207).
