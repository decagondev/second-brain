---
title: "E55 See and Steer the Context"
summary: "A person sees what each answer was built from, can change it, and can find out why something they expected wasn't used."
---

# E55: See and Steer the Context

> A person sees what each answer was built from, can change it, and can find out why something they expected wasn't used.

**Waves:** W54 · **Tickets:** 7 · **Stories:** 2

Planned 2026-11-26 after W52, with the user. The concept is in [22-see-steer-spend.md](../../docs/22-see-steer-spend.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, the AI label never comes off by itself, a rhythm only drafts and stops with its subject, a brief about a person draws only on what they published, lent access lasts one run and travels with its result, and a label never drops while the brain never changes who can open a source.

## User stories

- **US-55.1**: As *anyone*, I want a panel of what each answer used, with keep, drop, not-for-me and the memory it used, so that I can steer an answer instead of rewording the question. Delivered by SB-5501, SB-5502, SB-5503, SB-5504.
- **US-55.2**: As *anyone* or a *steward*, I want memory over MCP, "why not this document?" and a retrieval view on reported answers, so that I can find out why an answer missed something. Delivered by SB-5505, SB-5506, SB-5507.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S54.1 | Show the context | Context panel, Keep or drop for this conversation, Not for me, Memory used, shown | SB-5501, SB-5502, SB-5503, SB-5504 |
| S54.2 | Ask why | Memory over MCP, Why not this document?, Retrieval view for stewards | SB-5505, SB-5506, SB-5507 |

## Exit gate

- [ ] The context panel matches the trace, and a document the person can no longer open shows as a count
- [ ] A dropped or excluded document is never cited to that person, and still is to others
- [ ] "Why not" gives the same reply for a document the person can't open as for one that doesn't exist
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-5501](SB-5501-context-panel.md) | W54 / S54.1 | Context panel | LAW-01, LAW-26 |
| [SB-5502](SB-5502-keep-or-drop.md) | W54 / S54.1 | Keep or drop for this conversation | LAW-01 |
| [SB-5503](SB-5503-not-for-me.md) | W54 / S54.1 | Not for me | LAW-01, LAW-25 |
| [SB-5504](SB-5504-memory-used.md) | W54 / S54.1 | Memory used, shown | LAW-25 |
| [SB-5505](SB-5505-memory-over-mcp.md) | W54 / S54.2 | Memory over MCP | LAW-09, LAW-25 |
| [SB-5506](SB-5506-why-not.md) | W54 / S54.2 | Why not this document? | LAW-01, LAW-09 |
| [SB-5507](SB-5507-retrieval-view.md) | W54 / S54.2 | Retrieval view for stewards | LAW-01, LAW-26 |

## Decisions for the user

Approved by the user on 2026-11-26 ([22-see-steer-spend.md](../../docs/22-see-steer-spend.md#decisions-for-the-user)), every recommended decision: MCP clients read memory and propose new lines, which count only after the person confirms them in the web UI (SB-5505).
