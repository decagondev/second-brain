---
title: "SB-5801: Follow up in the thread"
summary: "A question asked in Slack or Mattermost can't be followed up in its thread."
---

# SB-5801: Follow up in the thread

| Field | Value |
|---|---|
| Epic | [E58 Carry the Conversation](README.md) |
| Wave / slice | W57 / S57.1 |
| Stories | US-58.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-cell(5)`](../../manual/sb-cell.5.md) |
| Laws | LAW-01, LAW-32 |
| Depends on | SB-2005, SB-2901 |
| Status | Done |

## Why

A question asked in Slack or Mattermost can't be followed up in its thread.

## Acceptance criteria

- [x] A reply in the thread under the brain's answer continues the same conversation, one `cv_…` per thread and person, the last 8 earlier questions sent, each turn re-authorised as the person who replied
- [x] A reply from someone else starts their own conversation in the same thread
- [x] A confidential answer in a channel thread goes to the person directly (LAW-32)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Thread to conversation in `slackapp` and `mmapp`
- [x] Labels in threads
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `Request.Thread` and `ask.ThreadConversation` make one conversation per thread and person (`cv_t…`, from the tenant, the person and the thread), new or continued, with the last 8 earlier questions sent as before. Slack: the chat connector hands each verified events delivery to `slackapp.App.Event` (`slack.Connector.Also`), which answers a reply whose `parent_user_id` is the bot (`DM.Bot`, `auth.test`) in a channel the bot is in (`DM.ChannelName`) through `App.FollowUp`: re-authorised as the replier, answered with `chat.postEphemeral` in the thread, and a confidential or restricted answer sent to their direct messages instead (LAW-32). Mattermost: an outgoing webhook to `POST /chat/mattermost/events` (`events_token`), the root post's author checked to be the bot, the same `FollowUp` with ephemeral posts. Tested in TestFollowUpInTheThread and TestThreadFollowUpsAreOnePersonsConversation.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
