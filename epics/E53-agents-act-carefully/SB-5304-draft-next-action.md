---
title: "SB-5304: Draft the next action"
summary: "An answer that recommends a ticket or a message leaves the person to write it from scratch."
---

# SB-5304: Draft the next action

| Field | Value |
|---|---|
| Epic | [E53 Agents That Act Carefully](README.md) |
| Wave / slice | W52 / S52.2 |
| Stories | US-53.2 |
| Manual | [`sb-actions(3)`](../../manual/sb-actions.3.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-04, LAW-16 |
| Depends on | SB-4301, SB-2105 |
| Status | Done |

## Why

An answer that recommends a ticket or a message leaves the person to write it from scratch.

## Acceptance criteria

- [x] **Draft it** prepares a Linear ticket or a Slack or Mattermost message from an answer, brief or review
- [x] The person edits it and presses **Send**, which goes out with their own token through the tool bus; agents never send
- [x] A cite they can't open isn't linked, and the sent item is audited

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Linear and chat drafts
- [x] Send through the tool bus
- [x] Tests
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: the user chose to send through the tool bus with the tenant's credentials rather than each person's own token, which would need per-person sign-in and a token store. Built: `Agent.DraftAction` at `POST /v1/actions/draft` `{trace, kind}` drafts from the caller's own answer, each cite checked again as them (`Recheck`), nothing sent or kept. A ticket comment is sent as any action, `POST /v1/actions` with `ticket-update`, so it waits for its approver and is audited. A message: `Stewards.SendMessage` at `POST /v1/spaces/NAME/messages` `{channel, text}`, for a member of the space, only in one of its `channels`, signed "Sent by EMAIL via Second Brain:", masked (SB-5102), audited `message_sent` before the bot posts it (`ask.ChannelPoster`: `mmapp.DM.Post` finds the channel by name in the team, `slackapp.DM.Post`; `Cell.chatPoster`). Web: `NextAction.tsx`, **Draft a ticket comment** and **Draft a message** under each answer. Tested in TestDraftTheNextAction, both TestPostInAChannel and web `nextaction.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
