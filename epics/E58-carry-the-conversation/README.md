---
title: "E58 Carry the Conversation"
summary: "A conversation follows a person from a chat thread to the web, to their other devices and to a colleague, and work they left unfinished finds them again."
---

# E58: Carry the Conversation

> A conversation follows a person from a chat thread to the web, to their other devices and to a colleague, and work they left unfinished finds them again.

**Waves:** W57 · **Tickets:** 7 · **Stories:** 2

Planned 2026-11-27 after W55, with the user. The concept is in [23-place-carry-reach.md](../../docs/23-place-carry-reach.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, the AI label never comes off by itself, a rhythm only drafts and stops with its subject, a brief about a person draws only on what they published, lent access lasts one run and travels with its result, a label never drops while the brain never changes who can open a source, and a model call stays in its region while a new vendor is an adapter against a recorded fake.

## User stories

- **US-58.1**: As *anyone*, I want to follow up in a chat thread, continue on the web, keep my conversations on every device and share a whole conversation, so that a conversation isn't stuck where it started. Delivered by SB-5801, SB-5802, SB-5803, SB-5804.
- **US-58.2**: As *anyone* or a *steward*, I want my open things shown when I start again, a reaction that saves a thread, and skills over MCP, so that unfinished work and useful threads aren't lost. Delivered by SB-5805, SB-5806, SB-5807.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S57.1 | Follow up anywhere | Follow up in the thread, Continue on the web, Your conversations on every device, Share a conversation | SB-5801, SB-5802, SB-5803, SB-5804 |
| S57.2 | Pick it up again | Where you left off, React to save, Skills over MCP | SB-5805, SB-5806, SB-5807 |

## Exit gate

- [ ] A follow-up in a chat thread continues the conversation as the person who replied, and only theirs
- [ ] A conversation continued on the web, on another device or by a colleague is checked again as that reader, answer by answer
- [ ] Synced conversations are the person's own: off by default, gone with forget, listed by nobody else
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-5801](SB-5801-follow-up-in-thread.md) | W57 / S57.1 | Follow up in the thread | LAW-01, LAW-32 |
| [SB-5802](SB-5802-continue-on-the-web.md) | W57 / S57.1 | Continue on the web | LAW-01, LAW-26 |
| [SB-5803](SB-5803-every-device.md) | W57 / S57.1 | Your conversations on every device | LAW-25, LAW-26 |
| [SB-5804](SB-5804-share-a-conversation.md) | W57 / S57.1 | Share a conversation | LAW-01, LAW-26, LAW-32 |
| [SB-5805](SB-5805-where-you-left-off.md) | W57 / S57.2 | Where you left off | LAW-01, LAW-26 |
| [SB-5806](SB-5806-react-to-save.md) | W57 / S57.2 | React to save | LAW-01, LAW-09 |
| [SB-5807](SB-5807-skills-over-mcp.md) | W57 / S57.2 | Skills over MCP | LAW-01, LAW-09 |

## Decisions for the user

Approved by the user on 2026-11-27 ([23-place-carry-reach.md](../../docs/23-place-carry-reach.md#decisions-for-the-user)), every recommended decision: conversations follow a person to every device only if they turn it on, holding questions, kept and dropped documents and trace ids for 30 days, gone with forget (SB-5803); one reaction (default `brain`) offers to save a thread, and nothing is read or saved until the person confirms in a DM (SB-5806).
