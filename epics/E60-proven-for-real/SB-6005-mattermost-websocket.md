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
| Status | Done |

## Why

Mattermost's outgoing webhooks carry no reactions, so react to save (SB-5806) and direct questions (SB-5805) are Slack only.

## Acceptance criteria

- [x] The Mattermost app keeps a websocket open as its bot (a small RFC 6455 client, standard library only) and reconnects with backoff
- [x] `reaction_added` with `chat.save_reaction` offers save or ask in a DM, and a DM to the bot is a direct question, answered as its sender
- [x] A reaction in a channel the bot isn't in does nothing; the webhook and `/brain` keep working without the websocket

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Websocket client and reconnects
- [x] Reaction and DM handling in `mmapp`
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `mmapp.Socket` is a standard-library RFC 6455 client (text, continuation, ping, close) on the bot's `/api/v4/websocket`, reconnecting with backoff, run by `Cell.Listen` from sb-api. `mmapp.App.Event` offers the save reaction in a DM with signed buttons, only for a channel the bot is in, and answers a DM to the bot as a direct question (`slackapp.App.DirectQuestion`, now shared); `reactPress` reads the thread only on the reactor's own signed press. Tested in TestWebsocketReactionsAndDirectQuestions and TestReactPressIsSignedAndOwned against a recorded Mattermost; a real server stays SB-2603's.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
