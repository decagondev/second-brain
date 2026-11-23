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
| Status | Done |

## Why

The Mattermost answer can't be shared to the channel the way the Slack one can.

## Acceptance criteria

- [x] The Mattermost answer gains a Share button (an interactive message action) that posts the question and the citation links only to the channel, the Slack rule (SB-308)
- [x] Only the asker can share, with their scope read again at the press
- [x] The Slack Share test passes against the Mattermost app

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Action endpoint
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: Done. The Mattermost answer carries a *Share in channel* message-attachment action to `POST /chat/mattermost/actions` (`mmapp.App.ActionURL`, from sb-cell(5) `base_url`). Mattermost doesn't sign action requests, so the context (trace, asker, `response_url`) carries an HMAC-SHA256 keyed with the command token, checked at the press; only the asker, mapped again to one active directory user, can share, and the channel gets `mmapp.Shared`: the citation links only, through the `response_url` as `in_channel`. The Slack Share tests run against it (`TestShareCarriesNoClaimText`, `TestOnlyTheAskerShares`), and `TestLAW01_MattermostSharesOnlyTheAskersCites` with a self-test that catches a share carrying the claims.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
