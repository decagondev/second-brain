---
title: "SB-5806: React to save"
summary: "A useful chat thread is saved for stewards only by a command few people know."
---

# SB-5806: React to save

| Field | Value |
|---|---|
| Epic | [E58 Carry the Conversation](README.md) |
| Wave / slice | W57 / S57.2 |
| Stories | US-58.2 |
| Manual | [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-09 |
| Depends on | SB-3703, SB-2005 |
| Status | Open |

## Why

A useful chat thread is saved for stewards only by a command few people know.

## Acceptance criteria

- [ ] sb-cell(5) `chat.save_reaction` (default `brain`); adding it to a thread sends the person one DM offering **Save for the stewards** or **Ask about this thread**
- [ ] Nothing is read beyond the reaction until they choose; the thread's text is data
- [ ] A thread in a channel the app isn't in is never read

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Reaction events in `slackapp` and `mmapp`
- [ ] DM with two actions
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
