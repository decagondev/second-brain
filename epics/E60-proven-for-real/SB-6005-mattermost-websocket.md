---
title: "SB-6005: Reactions and direct questions in Mattermost"
summary: "Mattermost's outgoing webhooks carry no reactions, so react to save (SB-5806) and direct questions (SB-5805) are Slack only."
---

# SB-6005: Reactions and direct questions in Mattermost

| Field | Value |
|---|---|
| Epic | [E60 Proven for Real](README.md) |
| Wave / slice | W59 / S59.2 |
| Stories | US-60.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-09 |
| Depends on | SB-5805, SB-5806, SB-2005 |
| Status | Open |

## Why

Mattermost's outgoing webhooks carry no reactions, so react to save (SB-5806) and direct questions (SB-5805) are Slack only.

## Acceptance criteria

- [ ] The Mattermost app keeps a websocket open as its bot (a small RFC 6455 client, standard library only) and reconnects with backoff
- [ ] `reaction_added` with `chat.save_reaction` offers save or ask in a DM, and a DM to the bot is a direct question, answered as its sender
- [ ] A reaction in a channel the bot isn't in does nothing; the webhook and `/brain` keep working without the websocket

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Websocket client and reconnects
- [ ] Reaction and DM handling in `mmapp`
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
