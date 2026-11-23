---
title: "E29 A Focused Context"
summary: "What the model reads stays short and on topic, and the operator can see when the index, the scopes, the eval cases or the prompts start to creep."
---

# E29: A Focused Context

> What the model reads stays short and on topic, and the operator can see when the index, the scopes, the eval cases or the prompts start to creep.

**Waves:** W28 · **Tickets:** 7 · **Stories:** 2

Proposed 2026-11-23 after W25 and accepted by the user the same day, with every recommended decision. The concept is in [13-fresh-tidy-focused.md](../../docs/13-fresh-tidy-focused.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, no new page, and nothing widens what anyone sees.

## User stories

- **US-29.1**: As an *asker*, I want the model to read only the recent part of a conversation and more than one document, so that answers stay on topic. Delivered by SB-2901, SB-2902.
- **US-29.2**: As an *admin*, *operator* or *steward*, I want to see scope patterns, eval cases, index leftovers, prompt size and reading lists that drift, so that creep is caught before answers get worse. Delivered by SB-2903, SB-2904, SB-2905, SB-2906, SB-2907.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S28.1 | Keep the model's context short | Conversations stay short, one document can't crowd out the rest, scope report | SB-2901, SB-2902, SB-2903 |
| S28.2 | Watch for creep | Eval cases that went stale, index leftovers, prompt growth, start here goes stale | SB-2904, SB-2905, SB-2906, SB-2907 |

## Exit gate

- [x] A conversation sends at most the last 8 earlier questions
- [x] At most 3 chunks of one document reach the model, and the eval suite doesn't regress
- [x] A scope typo and a too-wide pattern are named
- [x] Index leftovers are counted in health and pruned without touching what a source still lists
- [x] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-2901](SB-2901-short-conversations.md) | W28 / S28.1 | Conversations stay short | LAW-07 |
| [SB-2902](SB-2902-per-document-cap.md) | W28 / S28.1 | One document can't crowd out the rest | LAW-01 |
| [SB-2903](SB-2903-scope-report.md) | W28 / S28.1 | Scope report | LAW-16 |
| [SB-2904](SB-2904-stale-eval-cases.md) | W28 / S28.2 | Eval cases that went stale | — |
| [SB-2905](SB-2905-index-leftovers.md) | W28 / S28.2 | Index leftovers | LAW-13, LAW-11 |
| [SB-2906](SB-2906-prompt-growth.md) | W28 / S28.2 | Prompt growth watch | LAW-07 |
| [SB-2907](SB-2907-start-here-stale.md) | W28 / S28.2 | Start here goes stale | LAW-01 |

## Decisions for the user

Decided 2026-11-23 ([13-fresh-tidy-focused.md](../../docs/13-fresh-tidy-focused.md#decisions-for-the-user)): all of W26 to W28, in order; warn the asker when more than half of an answer's cites are stale (SB-2701); stewards retire a document from their own space's answers only, undoable and audited (SB-2801); owners' monthly note on by default, each person can turn it off (SB-2806); conversations send the last 8 questions (SB-2901).
