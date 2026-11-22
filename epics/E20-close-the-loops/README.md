---
title: "E20 Close the Loops"
summary: "W16's features work everywhere, and the brain reads the documents companies actually keep."
---

# E20: Close the Loops

> W16's features work everywhere, and the brain reads the documents companies actually keep.

**Waves:** W19 · **Tickets:** 5 · **Stories:** 3

Proposed 2026-11-22 after W16 and accepted by the user the same day, with every recommended decision. The concept is in [10-run-protect-close.md](../../docs/10-run-protect-close.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, and no new page.

## User stories

- **US-20.1**: As a *steward* or *manager*, I want notes in every cell and the handover brief on screen, so that W16's features work wherever the brain runs. Delivered by SB-2001, SB-2002.
- **US-20.2**: As an *asker*, I want answers from Word files, and notes that stay as narrow as their space, so that the brain knows what the share holds, and nothing more. Delivered by SB-2003, SB-2004.
- **US-20.3**: As an *asker* at a sovereign site, I want to ask from our own chat, so that I don't need Slack to use the brain. Delivered by SB-2005.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S19.1 | Finish W16 | Notes with barriers, the handover brief in the UI | SB-2001, SB-2002 |
| S19.2 | Read more, leak less | Office documents and HTML, notes that stay narrow | SB-2003, SB-2004 |
| S19.3 | Our own chat | A Mattermost entry point | SB-2005 |

## Exit gate

- [ ] A barrier cell publishes a note, and still refuses one across the barrier
- [ ] A manager reads a handover brief in the UI
- [ ] A share of Word files answers with cites
- [ ] A narrowed space lists its notes for review
- [ ] Mattermost asks answer under the asker's access
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-2001](SB-2001-notes-with-barriers.md) | W19 / S19.1 | Notes in cells with information barriers | LAW-04, LAW-22 |
| [SB-2002](SB-2002-handover-ui.md) | W19 / S19.1 | Handover brief in the UI | LAW-01, LAW-12 |
| [SB-2003](SB-2003-office-documents.md) | W19 / S19.2 | Word, OpenDocument and HTML on file shares | LAW-09, LAW-13 |
| [SB-2004](SB-2004-notes-stay-narrow.md) | W19 / S19.2 | Notes stay narrow | LAW-19, LAW-21 |
| [SB-2005](SB-2005-mattermost.md) | W19 / S19.3 | Mattermost entry point | LAW-01, LAW-07, LAW-12 |

## Decisions for the user

Decided 2026-11-22 ([10-run-protect-close.md](../../docs/10-run-protect-close.md#decisions-for-the-user)): all of W17 to W19, in order; a 3-of-5 recovery kit on paper (SB-1902); sealing on by default (SB-1901); Mattermost (SB-2005).
