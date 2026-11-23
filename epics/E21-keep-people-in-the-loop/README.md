---
title: "E21 Keep People in the Loop"
summary: "When something needs a person, they hear about it, and nothing they hear tells them more than they could open."
---

# E21: Keep People in the Loop

> When something needs a person, they hear about it, and nothing they hear tells them more than they could open.

**Waves:** W20 · **Tickets:** 6 · **Stories:** 3

Proposed 2026-11-23 after W19 and accepted by the user the same day, with every recommended decision. The concept is in [11-loop-own-fit.md](../../docs/11-loop-own-fit.md). Every ticket keeps to the bloat rules there: no new service the cell must run, no new store, no new third-party Go module, at most one setting a feature, and no new page.

## User stories

- **US-21.1**: As a *steward* or *asker*, I want to be told when a question or a reply waits for me, so that questions don't sit unanswered and replies aren't missed. Delivered by SB-2101, SB-2102.
- **US-21.2**: As an *asker*, I want to hear what changed in my spaces and in my saved questions' sources, so that I don't keep relying on something that moved. Delivered by SB-2103, SB-2104.
- **US-21.3**: As an *asker* or *manager*, I want to share from Mattermost and brief someone by name, so that the chat app and the handover tab work like the rest. Delivered by SB-2105, SB-2106.

## Slices

| Slice | Name | Delivers | Tickets |
|---|---|---|---|
| S20.1 | Questions and replies | Stewards and askers notified | SB-2101, SB-2102 |
| S20.2 | What changed, for you | Weekly digest, watched saved questions | SB-2103, SB-2104 |
| S20.3 | Chat and handover parity | Share in Mattermost, handover by name | SB-2105, SB-2106 |

## Exit gate

- [ ] A steward who left the group between question and send is told nothing
- [ ] No notification carries question, answer, cite or document text
- [ ] A digest never names a document the person can't open now
- [ ] Mattermost Share passes the Slack Share test
- [ ] Every law test still green; no new store, service or Go module

## Tickets

| Ticket | Wave / slice | Title | Laws |
|---|---|---|---|
| [SB-2101](SB-2101-stewards-notified.md) | W20 / S20.1 | Stewards hear about new questions | LAW-21, LAW-07 |
| [SB-2102](SB-2102-askers-notified.md) | W20 / S20.1 | Askers hear about replies | LAW-01, LAW-07 |
| [SB-2103](SB-2103-weekly-digest.md) | W20 / S20.2 | Weekly "what changed" digest | LAW-01, LAW-13 |
| [SB-2104](SB-2104-watch-saved.md) | W20 / S20.2 | Watch a saved question | LAW-01, LAW-07 |
| [SB-2105](SB-2105-mattermost-share.md) | W20 / S20.3 | Share in Mattermost | LAW-01 |
| [SB-2106](SB-2106-handover-by-name.md) | W20 / S20.3 | Handover by name | LAW-01 |

## Decisions for the user

Decided 2026-11-23 ([11-loop-own-fit.md](../../docs/11-loop-own-fit.md#decisions-for-the-user)): all of W20 to W22, in order; notifications by Slack or Mattermost direct message with email as the fallback (SB-2101, SB-2307); English, German and French (SB-2301); read-only API tokens, 90 days at most (SB-2306).
