---
title: "SB-5804: Share a conversation"
summary: "Only one answer at a time can be shared."
---

# SB-5804: Share a conversation

| Field | Value |
|---|---|
| Epic | [E58 Carry the Conversation](README.md) |
| Wave / slice | W57 / S57.1 |
| Stories | US-58.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-26, LAW-32 |
| Depends on | SB-1703, SB-5405 |
| Status | Done |

## Why

Only one answer at a time can be shared.

## Acceptance criteria

- [x] **Share conversation** sends a whole conversation to a colleague like a shared answer
- [x] The reader sees each answer checked again as them; an answer they can't open every cite of is left out and counted
- [x] The highest label of what they see goes with it, a confidential share is never posted to a channel, and revoking removes it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Conversation shares
- [x] Web share and reader view
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. `ask.ConversationShares` (Share, Open, Revoke): a `Share` with `Traces` and `Questions`, audited, opened by the reader with each answer checked as them, one they can't open every cite of left out and counted (`left`), the highest label of what they see (`label`); `ask.ShareRevoker` expires it at once. `POST /v1/me/conversations/{id}/share`, `GET|DELETE /v1/conversation-shares/{id}`, the web's `/c/sh_…`; a conversation share never opens as a question share. Tested in TestShareAConversation and TestCarryTheConversationThroughARunningCell.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
