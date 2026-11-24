---
title: "E43 Work That Comes to You"
summary: "Each person sees in one place what waits on them, has recurring questions answered for them, and keeps context for their work, all checked as them."
---

# E43: Work That Comes to You

> Each person sees in one place what waits on them, has recurring questions answered for them, and keeps context for their work, all checked as them.

**Waves:** W42 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-24 after W40. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, and every long run has a ceiling.

## User stories

- **US-43.1**: As *anyone*, I want one inbox, routines and a memory I control, so that what needs me finds me and answers fit me. Delivered by SB-4301, SB-4302, SB-4303.
- **US-43.2**: As *anyone*, I want collections, a record of what agents did as me, agents in my other tools and exact numbers from sheets, so that I can trust and reuse the brain in my own work. Delivered by SB-4304, SB-4305, SB-4306, SB-4307.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S42.1 | One inbox | One inbox, my routines, memory I control | SB-4301, SB-4302, SB-4303 |
| S42.2 | Context for the work | Collections, what agents did as me, agents over MCP, numbers from tables | SB-4304, SB-4305, SB-4306, SB-4307 |

## Exit gate

- [ ] Every inbox item, routine answer and collection item is checked as the reader when shown
- [ ] A memory never holds document text or reaches another person's ask (LAW-25 green)
- [ ] Every run made as a person appears on their Agent runs
- [ ] A computed number in an answer always comes from the table tool, with its cells cited
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-4301](SB-4301-one-inbox.md) | W42 / S42.1 | One inbox | LAW-01, LAW-07, LAW-26 |
| [SB-4302](SB-4302-routines.md) | W42 / S42.1 | My routines | LAW-01, LAW-12, LAW-26 |
| [SB-4303](SB-4303-memory.md) | W42 / S42.1 | Memory I control | LAW-25, LAW-07, LAW-09 |
| [SB-4304](SB-4304-collections.md) | W42 / S42.2 | Collections | LAW-01, LAW-13 |
| [SB-4305](SB-4305-agent-runs.md) | W42 / S42.2 | What agents did as me | LAW-01, LAW-11 |
| [SB-4306](SB-4306-agents-over-mcp.md) | W42 / S42.2 | Agents over MCP | LAW-01, LAW-10 |
| [SB-4307](SB-4307-numbers-from-tables.md) | W42 / S42.2 | Numbers from tables | LAW-01, LAW-14 |

## Decisions for the user

Waiting on the user ([18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md#decisions-for-the-user)). Recommended: all of W41 to W43, in order; memory on, holding only what each person writes, shown, exported and forgettable, with an admin switch (SB-4303).
