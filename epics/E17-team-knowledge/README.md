---
title: "E17 Team Knowledge"
summary: "Ten small features that let a team hand knowledge to each other through the brain, and keep it current, on one sovereign machine."
---

# E17: Team Knowledge

> Ten small features that let a team hand knowledge to each other through the brain, and keep it current, on one sovereign machine.

**Waves:** W16 · **Tickets:** 10 · **Stories:** 5

Proposed 2026-11-22 after W15 and accepted by the user the same day, with every recommended decision. The research and the concept are in [09-team-knowledge.md](../../docs/09-team-knowledge.md). Every ticket keeps to the bloat rules there: no new service the cell must run (the model container is an optional profile), no new store, no new third-party Go module, at most one manifest field a feature, and one page in the web UI.

## User stories

- **US-17.1**: As an *asker*, I want to find documents fast, see who owns them and keep or pass on a good question, so that I spend less time hunting and asking around. Delivered by SB-1701, SB-1702, SB-1703.
- **US-17.2**: As an *asker*, I want a question nothing answers to reach the people who know, and as a *steward*, I want my answer to become a document, so that each answer is given once. Delivered by SB-1704, SB-1705.
- **US-17.3**: As a *new joiner* or a *successor*, I want a place to start and a brief of what my predecessor owned, so that knowledge survives people moving. Delivered by SB-1706, SB-1707.
- **US-17.4**: As an *asker*, I want to see what changed in my spaces, and as a *steward*, I want to see which relied-on documents are going stale, so that answers stay current. Delivered by SB-1708, SB-1709.
- **US-17.5**: As an *operator* of a sovereign cell, I want answers and embeddings from a model on the box, so that no question leaves the machine. Delivered by SB-1710.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S16.1 | Find | Search without the model, owners on cites, saved and shared questions | SB-1701, SB-1702, SB-1703 |
| S16.2 | Ask and capture | Questions to stewards, and replies published as notes | SB-1704, SB-1705 |
| S16.3 | Onboard and hand over | Start here per space, and a handover brief | SB-1706, SB-1707 |
| S16.4 | Keep it current | What changed, and documents to review | SB-1708, SB-1709 |
| S16.5 | On the box | A local model and embedder, pinned and gated | SB-1710 |

## Exit gate

- [x] A search answers with no model call, and returns only documents the asker can open now
- [ ] A question nothing answered reaches a steward, and their published note answers the next ask with a cite
- [ ] A new member sees their space's start list, and a successor gets a cited handover brief
- [ ] Stewards see stale, relied-on documents; askers see what changed, each item checked live
- [ ] The self-hosted demo answers with the `model` profile and nothing leaves the box (`make selfhost-smoke`)
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-1701](SB-1701-search-without-the-model.md) | W16 / S16.1 | Search without the model | LAW-01, LAW-03, LAW-14 |
| [SB-1702](SB-1702-who-to-ask.md) | W16 / S16.1 | Who to ask: owners on citations | LAW-01, LAW-12, LAW-22 |
| [SB-1703](SB-1703-save-and-share-a-question.md) | W16 / S16.1 | Save and share a question | LAW-01, LAW-02, LAW-07 |
| [SB-1704](SB-1704-ask-the-stewards.md) | W16 / S16.2 | Ask the stewards when nothing answers | LAW-01, LAW-04, LAW-07, LAW-21 |
| [SB-1705](SB-1705-replies-become-notes.md) | W16 / S16.2 | Answer once: replies become notes | LAW-04, LAW-05, LAW-09, LAW-13, LAW-15 |
| [SB-1706](SB-1706-start-here.md) | W16 / S16.3 | Start here: a reading list per space | LAW-01, LAW-03, LAW-16 |
| [SB-1707](SB-1707-handover-brief.md) | W16 / S16.3 | Handover brief for a successor | LAW-01, LAW-09, LAW-12, LAW-14 |
| [SB-1708](SB-1708-what-changed.md) | W16 / S16.4 | What changed in my spaces | LAW-01, LAW-13 |
| [SB-1709](SB-1709-documents-to-review.md) | W16 / S16.4 | Documents to review for stewards | LAW-01, LAW-14, LAW-16 |
| [SB-1710](SB-1710-local-model-on-the-box.md) | W16 / S16.5 | A local model on the box | LAW-02, LAW-06, LAW-07, LAW-13 |

## Decisions for the user

Decided 2026-11-22 ([09-team-knowledge.md](../../docs/09-team-knowledge.md#decisions-for-the-user)): all of W16; a steward's publish is a note's approval, bound to its hash (SB-1705); llama.cpp's server (SB-1710); a handover brief for the person's manager, a steward of one of their spaces, or the person (SB-1707).
