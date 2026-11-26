---
title: "E56 Keep Answering, Spend Less"
summary: "A provider failing doesn't stop the brain, and the same quality costs less."
---

# E56: Keep Answering, Spend Less

> A provider failing doesn't stop the brain, and the same quality costs less.

**Waves:** W55 · **Tickets:** 6 · **Stories:** 2

Planned 2026-11-26 after W52, with the user. The concept is in [22-see-steer-spend.md](../../docs/22-see-steer-spend.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, the AI label never comes off by itself, a rhythm only drafts and stops with its subject, a brief about a person draws only on what they published, lent access lasts one run and travels with its result, and a label never drops while the brain never changes who can open a source.

## User stories

- **US-56.1**: As *anyone* or an *admin*, I want a gated fallback model, passages when no model is up and reuse of an unchanged answer, so that the brain keeps answering when a provider fails. Delivered by SB-5601, SB-5602, SB-5603.
- **US-56.2**: As an *admin* or *finance*, I want batching, prompt caching and a line showing what they saved, so that the same quality costs less and I can see by how much. Delivered by SB-5604, SB-5605, SB-5606.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S55.1 | Keep answering | Fall back to a gated model, Passages when the model is down, Answered before, checked now | SB-5601, SB-5602, SB-5603 |
| S55.2 | Spend less | Batch what can wait, Cache the stable part, What it saved | SB-5604, SB-5605, SB-5606 |

## Exit gate

- [ ] An ungated fallback is never used; with every provider down an ask returns passages and no generated text
- [ ] A reused answer is checked again as the new asker, and a changed cite forces a fresh one
- [ ] No passage, memory or question is ever in a cached prompt prefix, and the savings add up from traces
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-5601](SB-5601-gated-fallback.md) | W55 / S55.1 | Fall back to a gated model | LAW-03, LAW-06 |
| [SB-5602](SB-5602-passages-when-down.md) | W55 / S55.1 | Passages when the model is down | LAW-01, LAW-14 |
| [SB-5603](SB-5603-answer-reuse.md) | W55 / S55.1 | Answered before, checked now | LAW-01, LAW-26 |
| [SB-5604](SB-5604-batch-what-can-wait.md) | W55 / S55.2 | Batch what can wait | LAW-05, LAW-16 |
| [SB-5605](SB-5605-prompt-caching.md) | W55 / S55.2 | Cache the stable part | LAW-02, LAW-07 |
| [SB-5606](SB-5606-what-it-saved.md) | W55 / S55.2 | What it saved | LAW-11 |

## Decisions for the user

Approved by the user on 2026-11-26 ([22-see-steer-spend.md](../../docs/22-see-steer-spend.md#decisions-for-the-user)), every recommended decision: answer reuse is on for 7 days, same question and space, every cite unchanged and open to the asker, always marked with **Ask fresh** (SB-5603); a fallback model is used only if it passed the same eval gate, otherwise passages only (SB-5601, SB-5602); scheduled briefs, routines, checks and evals go through batch APIs with results by the next working morning (SB-5604).
