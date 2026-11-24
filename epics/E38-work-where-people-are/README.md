---
title: "E38 Work Where People Are"
summary: "The brain is in reach from wherever people work, and admins see how it's used."
---

# E38: Work Where People Are

> The brain is in reach from wherever people work, and admins see how it's used.

**Waves:** W37 · **Tickets:** 6 · **Stories:** 2

Proposed 2026-11-24 after W34 and accepted by the user the same day, with every recommended decision. The concept is in [16-trust-capture-reach.md](../../docs/16-trust-capture-reach.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-38.1**: As a *person using other tools*, I want to use the brain from an AI tool, ask about a file, and have admins see every token, so that the brain is where I work. Delivered by SB-3801, SB-3802, SB-3803.
- **US-38.2**: As an *admin* or *person using the brain*, I want to see adoption, get a daily brief, and search from the address bar, so that we know it's used and it's one step away. Delivered by SB-3804, SB-3805, SB-3806.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S37.1 | From other tools | Use the brain from other AI tools, ask about a file, every token in one place | SB-3801, SB-3802, SB-3803 |
| S37.2 | Know it's used | Adoption, daily brief, search from the address bar | SB-3804, SB-3805, SB-3806 |

## Exit gate

- [ ] MCP tools are read-only, off by default, and stop with a revoked token
- [ ] An attached file is never kept in the index or the trace
- [ ] No token is ever shown after it is made
- [ ] Adoption names no one; no space row under 5 people
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-3801](SB-3801-mcp.md) | W37 / S37.1 | Use the brain from other AI tools | LAW-01, LAW-11, LAW-12 |
| [SB-3802](SB-3802-ask-about-file.md) | W37 / S37.1 | Ask about a file | LAW-07, LAW-09, LAW-14 |
| [SB-3803](SB-3803-admin-tokens.md) | W37 / S37.1 | Every token in one place | LAW-11, LAW-12 |
| [SB-3804](SB-3804-adoption.md) | W37 / S37.2 | Adoption | LAW-07 |
| [SB-3805](SB-3805-daily-brief.md) | W37 / S37.2 | Daily brief | LAW-07, LAW-12 |
| [SB-3806](SB-3806-opensearch.md) | W37 / S37.2 | Search from the address bar | — |

## Decisions for the user

Decided 2026-11-24 ([16-trust-capture-reach.md](../../docs/16-trust-capture-reach.md#decisions-for-the-user)): all of W35 to W37, in order; a pinned answer shows first with every cite checked again as the asker (SB-3601); MCP read-only with the person's own token, off until the admin turns it on (SB-3801); one file of up to 2 MB per ask, never kept (SB-3802); a report is sent by the asker's choice and names them (SB-3606).
