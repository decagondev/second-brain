---
title: "SB-3205: Answer time watch"
summary: "Nobody watches how long answers take."
---

# SB-3205: Answer time watch

| Field | Value |
|---|---|
| Epic | [E32 Run Without Surprises](README.md) |
| Wave / slice | W31 / S31.2 |
| Stories | US-32.2 |
| Manual | [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-07 |
| Depends on | SB-2906 |
| Status | Done |

## Why

Nobody watches how long answers take.

## Acceptance criteria

- [x] The usage ledger records each model call's duration; health and the weekly report show this week's and last week's 95th percentile of seconds
- [x] It warns at more than 1.5 times last week's
- [x] A doubled week warns; a flat one doesn't

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Ledger
- [x] Health and report
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Ledger.Observe records sb_ask_seconds as answer_ms; health.AnswerFrom gives answer_time with this week's and last week's 95th percentile, warning over 1.5 times last week; the weekly report has a Seconds per answer row and Checks has answer_time. Tests: health/answer_test.go, usage/localfs/answer_test.go.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
