---
title: "SB-3805: Daily brief"
summary: "The digest comes once a week, which is too slow for some people."
---

# SB-3805: Daily brief

| Field | Value |
|---|---|
| Epic | [E38 Work Where People Are](README.md) |
| Wave / slice | W37 / S37.2 |
| Stories | US-38.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07, LAW-12 |
| Depends on | SB-2103 |
| Status | Done |

## Why

The digest comes once a week, which is too slow for some people.

## Acceptance criteria

- [x] Notification settings take `digest: daily`, which sends a short message each working morning at 08:00 instead of Monday's
- [x] It holds what changed in their spaces and their watched questions
- [x] A quiet day sends nothing; weekly people are unchanged

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notify
- [x] Settings
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `notify.Prefs.Digest` (`daily`), `ErrDigest`, `WorkingMorning`, `Job.daily`, `Job.dailyDigests`, `Job.watched` (shared with the weekly part), the `daily` clock part; `Job.Changes` takes the days. `PUT /v1/me/notifications` takes `digest`; the Questions tab's settings have **Daily, each working morning**. TestDailyBrief, TestWorkingMorning, TestPrefsDigest, TestDailyBriefSetting, web brief.test.tsx; LAW-01's digest law runs with the days.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
