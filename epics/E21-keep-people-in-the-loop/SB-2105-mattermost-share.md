---
title: "SB-2105: Share in Mattermost"
summary: "The Mattermost answer can't be shared to the channel the way the Slack one can."
---

# SB-2105: Share in Mattermost

| Field | Value |
|---|---|
| Epic | [E21 Keep People in the Loop](README.md) |
| Wave / slice | W20 / S20.3 |
| Stories | US-21.3 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Open |

## Why

The Mattermost answer can't be shared to the channel the way the Slack one can.

## Acceptance criteria

- [ ] The Mattermost answer gains a Share button (an interactive message action) that posts the question and the citation links only to the channel, the Slack rule (SB-308)
- [ ] Only the asker can share, with their scope read again at the press
- [ ] The Slack Share test passes against the Mattermost app

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Action endpoint
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
