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
| Status | Done |

## Why

A steward finds a new question only by opening the steward screen; questions wait for someone to look.

## Acceptance criteria

- [x] A `notify` job in `sb-jobs` runs every five minutes over steward questions created since its last run, and sends each space's current stewards (their groups at send time) one message: the count, the space and a link to the steward screen
- [x] Never the question's text, an answer or a cite in any message or log; a steward removed from the group between question and send gets nothing
- [x] Messages go as a direct message through the Slack or Mattermost bot (the Slack bot gains `chat:write` and `im:write`), and a person can turn them off

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notify job and its cursor
- [x] Slack and Mattermost DM senders
- [x] Opt-out
- [x] Law tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: Done. `internal/core/notify` is the notify job, run by `sb-jobs` every five minutes (sb-runtime(7)): it counts the new questions of each space and tells each of that space's stewards as the directory has them at send time, active only, with a link to the steward screen and never the text. `slackapp.DM` and `mmapp.DM` send direct messages; `wire.Cell.Notifier` picks Mattermost's bot, else Slack's. `GET|PUT /v1/me/notifications` keeps a person's settings (`pref` records, erased with the person, never expired); the Questions tab has the checkboxes. Law tests: `TestLAW21_NotificationsGoToTheStewardsThereAreNow`, `TestLAW07_NotificationsSayNothingOfWhatWasAsked`, self-test `TestSelf_NotifyLaw`; running cell: `TestNotificationsThroughARunningCell`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
