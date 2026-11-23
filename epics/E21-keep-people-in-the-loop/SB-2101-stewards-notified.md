---
title: "SB-2101: Stewards hear about new questions"
summary: "A steward finds a new question only by opening the steward screen; questions wait for someone to look."
---

# SB-2101: Stewards hear about new questions

| Field | Value |
|---|---|
| Epic | [E21 Keep People in the Loop](README.md) |
| Wave / slice | W20 / S20.1 |
| Stories | US-21.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-cell(5)`](../../manual/sb-cell.5.md), [`sb-runtime(7)`](../../manual/sb-runtime.7.md) |
| Laws | LAW-21, LAW-07 |
| Depends on | — |
| Status | Open |

## Why

A steward finds a new question only by opening the steward screen; questions wait for someone to look.

## Acceptance criteria

- [ ] A `notify` job in `sb-jobs` runs every five minutes over steward questions created since its last run, and sends each space's current stewards (their groups at send time) one message: the count, the space and a link to the steward screen
- [ ] Never the question's text, an answer or a cite in any message or log; a steward removed from the group between question and send gets nothing
- [ ] Messages go as a direct message through the Slack or Mattermost bot (the Slack bot gains `chat:write` and `im:write`), and a person can turn them off

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Notify job and its cursor
- [ ] Slack and Mattermost DM senders
- [ ] Opt-out
- [ ] Law tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
