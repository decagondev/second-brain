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
| Status | Open |

## Why

A question asked in Slack or Mattermost can't be followed up in its thread.

## Acceptance criteria

- [ ] A reply in the thread under the brain's answer continues the same conversation, one `cv_…` per thread and person, the last 8 earlier questions sent, each turn re-authorised as the person who replied
- [ ] A reply from someone else starts their own conversation in the same thread
- [ ] A confidential answer in a channel thread goes to the person directly (LAW-32)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Thread to conversation in `slackapp` and `mmapp`
- [ ] Labels in threads
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
