---
title: "E57 Right Model, Right Place"
summary: "Every call goes to the cheapest gated model that fits it, in the region the company allows, people come first when a provider is busy, and work that can wait is batched on every provider without holding a run open."
---

# E57: Right Model, Right Place

> Every call goes to the cheapest gated model that fits it, in the region the company allows, people come first when a provider is busy, and work that can wait is batched on every provider without holding a run open.

**Waves:** W56 · **Tickets:** 6 · **Stories:** 2

Planned 2026-11-27 after W55, with the user. The concept is in [23-place-carry-reach.md](../../docs/23-place-carry-reach.md). Every ticket keeps to the rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, nothing widens what anyone sees, a person approves every write, the tree only narrows, agents are definitions in the release bundle, stored results are checked when read, every long run has a ceiling, one level of consultation, the AI label never comes off by itself, a rhythm only drafts and stops with its subject, a brief about a person draws only on what they published, lent access lasts one run and travels with its result, a label never drops while the brain never changes who can open a source, and a model call stays in its region while a new vendor is an adapter against a recorded fake.

## User stories

- **US-57.1**: As an *admin*, I want batches collected on a later tick, OpenAI batches and people served first when a provider is busy, so that work that can wait costs less without slowing anyone down. Delivered by SB-5701, SB-5702, SB-5703.
- **US-57.2**: As an *admin* or *security*, I want a deeper tier only where it helps, model calls kept in our region and cache hits I can see, so that each call goes to the right model in the right place. Delivered by SB-5704, SB-5705, SB-5706.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S56.1 | Batches and busy providers | Collect batches later, OpenAI batches, People first when the provider is busy | SB-5701, SB-5702, SB-5703 |
| S56.2 | The right model | Think harder only when it helps, Keep model calls in region, Cache hits you can see | SB-5704, SB-5705, SB-5706 |

## Exit gate

- [ ] A batch submitted on one tick is collected on a later one, a restart loses nothing, and anything not back by 06:00 runs normally
- [ ] With `model.region` set no call of any tier leaves the region, and a provider with no region is never used (LAW-33 green)
- [ ] An ungated deep tier is never used; in a rate-limit storm asks finish before scheduled work
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-5701](SB-5701-collect-batches-later.md) | W56 / S56.1 | Collect batches later | LAW-05, LAW-07 |
| [SB-5702](SB-5702-openai-batches.md) | W56 / S56.1 | OpenAI batches | LAW-05, LAW-07 |
| [SB-5703](SB-5703-people-first-when-busy.md) | W56 / S56.1 | People first when the provider is busy | LAW-03, LAW-14 |
| [SB-5704](SB-5704-think-harder.md) | W56 / S56.2 | Think harder only when it helps | LAW-06, LAW-14 |
| [SB-5705](SB-5705-keep-calls-in-region.md) | W56 / S56.2 | Keep model calls in region | LAW-03, LAW-33 |
| [SB-5706](SB-5706-cache-hits.md) | W56 / S56.2 | Cache hits you can see | LAW-07, LAW-11 |

## Decisions for the user

Approved by the user on 2026-11-27 ([23-place-carry-reach.md](../../docs/23-place-carry-reach.md#decisions-for-the-user)), every recommended decision: batches are submitted on one tick and collected on a later one, with anything not back by 06:00 on the next working morning run normally, and OpenAI batches only for providers that declare `model.batch` (SB-5701, SB-5702); an optional `model.region` that no model call of any tier leaves, and a provider with no region is never used (SB-5705, LAW-33); a deeper tier only for research synthesis, tables, consultations and one retry after a failed citation check, and only while gated (SB-5704).
