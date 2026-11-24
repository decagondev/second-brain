---
title: "E30 Show the Evidence"
summary: "An asker can check an answer without leaving it, and the answer reads in their language."
---

# E30: Show the Evidence

> An asker can check an answer without leaving it, and the answer reads in their language.

**Waves:** W29 · **Tickets:** 6 · **Stories:** 2

Proposed 2026-11-24 after W28 and accepted by the user the same day, with every recommended decision. The concept is in [14-evidence-gaps-surprises.md](../../docs/14-evidence-gaps-surprises.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-30.1**: As an *asker*, I want to read the cited passage and its section, in an answer written in my language, so that I can check an answer where I read it. Delivered by SB-3001, SB-3002, SB-3003.
- **US-30.2**: As an *asker*, I want to ask about one document, be told when an answer rests on one document, and open a cite at its lines, so that I know how much an answer rests on. Delivered by SB-3004, SB-3005, SB-3006.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S29.1 | See what it rests on | Show the passage, section on the cite, answers in the asker's language | SB-3001, SB-3002, SB-3003 |
| S29.2 | Narrow it and say how thin it is | Ask about this document, one-source warning, open at the cited lines | SB-3004, SB-3005, SB-3006 |

## Exit gate

- [ ] A passage is checked live on every open and refused once access is gone
- [ ] A cite names its section, and a rebuild fills it in
- [ ] A German ask gets German claims with the same cites, and the eval cases pass
- [ ] A document filter only narrows
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-3001](SB-3001-show-passage.md) | W29 / S29.1 | Show the passage | LAW-01, LAW-07, LAW-09 |
| [SB-3002](SB-3002-section-on-cite.md) | W29 / S29.1 | Section on the cite | LAW-09, LAW-13 |
| [SB-3003](SB-3003-answer-language.md) | W29 / S29.1 | Answers in the asker's language | LAW-14, LAW-07 |
| [SB-3004](SB-3004-ask-about-document.md) | W29 / S29.2 | Ask about this document | LAW-01 |
| [SB-3005](SB-3005-one-source-warning.md) | W29 / S29.2 | One-source warning | LAW-01, LAW-14 |
| [SB-3006](SB-3006-cited-lines.md) | W29 / S29.2 | Open at the cited lines | — |

## Decisions for the user

Decided 2026-11-24 ([14-evidence-gaps-surprises.md](../../docs/14-evidence-gaps-surprises.md#decisions-for-the-user)): all of W29 to W31, in order; answers in the language picked in the UI, or the manifest's in chat (SB-3003); passages of up to 600 characters (SB-3001); owners mark only their own documents reviewed (SB-3106); admins alone set the notice, ending within 7 days (SB-3207).
