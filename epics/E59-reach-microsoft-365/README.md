---
title: "E59 Reach Microsoft 365"
summary: "A company on Microsoft 365 gets the same brain as one on Google Workspace: documents with their sensitivity labels, groups behind every check, Teams as a place to ask, and calendars, transcripts and shared mailboxes feeding the same agents."
---

# E59: Reach Microsoft 365

> A company on Microsoft 365 gets the same brain as one on Google Workspace: documents with their sensitivity labels, groups behind every check, Teams as a place to ask, and calendars, transcripts and shared mailboxes feeding the same agents.

**Waves:** W58 · **Tickets:** 7 · **Stories:** 2

Planned 2026-11-27 after W55, with the user. The concept is in [23-place-carry-reach.md](../../docs/23-place-carry-reach.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, the AI label never comes off by itself, a rhythm only drafts and stops with its subject, a brief about a person draws only on what they published, lent access lasts one run and travels with its result, a label never drops while the brain never changes who can open a source, and a model call stays in its region while a new vendor is an adapter against a recorded fake.

## User stories

- **US-59.1**: As an *admin* or *security*, I want SharePoint and OneDrive with their Purview labels and Entra groups behind every check, so that a Microsoft 365 company's documents are cited exactly as their sharing allows. Delivered by SB-5901, SB-5902, SB-5903.
- **US-59.2**: As *anyone* or a *steward*, I want to ask in Teams, prep meetings from Outlook, cite Teams transcripts and start on a shared mailbox's mail, so that the brain works where a Microsoft 365 company works. Delivered by SB-5904, SB-5905, SB-5906, SB-5907.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S58.1 | Documents and groups | SharePoint and OneDrive, Purview labels, Entra groups and people | SB-5901, SB-5902, SB-5903 |
| S58.2 | Teams, calendar and mail | Ask in Teams, Outlook calendar for meeting prep, Teams meeting transcripts, Start on a new email | SB-5904, SB-5905, SB-5906, SB-5907 |

## Exit gate

- [ ] The recorded Graph fake syncs with its permissions and sharing links, and a delete removes every chunk at the next sync
- [ ] An unmapped Purview label counts as restricted; a confidential answer in Teams goes to the person directly
- [ ] Only a shared mailbox a space names is ever read, and nothing is sent without a person
- [ ] Every law test still green; no new store, service or Go module; real-tenant runs stay In review

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-5901](SB-5901-sharepoint-onedrive.md) | W58 / S58.1 | SharePoint and OneDrive | LAW-01, LAW-13 |
| [SB-5902](SB-5902-purview-labels.md) | W58 / S58.1 | Purview labels | LAW-03, LAW-32 |
| [SB-5903](SB-5903-entra.md) | W58 / S58.1 | Entra groups and people | LAW-01, LAW-12 |
| [SB-5904](SB-5904-ask-in-teams.md) | W58 / S58.2 | Ask in Teams | LAW-01, LAW-32 |
| [SB-5905](SB-5905-outlook-calendar.md) | W58 / S58.2 | Outlook calendar for meeting prep | LAW-01, LAW-30 |
| [SB-5906](SB-5906-teams-transcripts.md) | W58 / S58.2 | Teams meeting transcripts | LAW-01, LAW-09 |
| [SB-5907](SB-5907-start-on-email.md) | W58 / S58.2 | Start on a new email | LAW-09, LAW-16 |

## Decisions for the user

Approved by the user on 2026-11-27 ([23-place-carry-reach.md](../../docs/23-place-carry-reach.md#decisions-for-the-user)), every recommended decision: SharePoint and OneDrive, Purview labels with unmapped labels treated as `restricted`, Entra, Teams, Outlook calendar, Teams transcripts and shared-mailbox triggers only, built against recorded fakes with real-tenant runs In review (ADR-0023).
