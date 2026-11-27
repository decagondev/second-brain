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
| Status | Open |

## Why

Only one answer at a time can be shared.

## Acceptance criteria

- [ ] **Share conversation** sends a whole conversation to a colleague like a shared answer
- [ ] The reader sees each answer checked again as them; an answer they can't open every cite of is left out and counted
- [ ] The highest label of what they see goes with it, a confidential share is never posted to a channel, and revoking removes it

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Conversation shares
- [ ] Web share and reader view
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
