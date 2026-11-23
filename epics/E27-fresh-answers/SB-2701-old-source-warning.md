---
title: "SB-2701: Old-source warning"
summary: "An answer that rests mostly on old documents looks just like a fresh one."
---

# SB-2701: Old-source warning

| Field | Value |
|---|---|
| Epic | [E27 Fresh Answers](README.md) |
| Wave / slice | W26 / S26.1 |
| Stories | US-27.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-14 |
| Depends on | — |
| Status | Done |

## Why

An answer that rests mostly on old documents looks just like a fresh one.

## Acceptance criteria

- [x] When more than half of an answer's cites are stale, the response carries `notice_stale` and the UI, Slack and Mattermost show one warning line first
- [x] Only cites the asker can see are counted
- [x] The warning shows at 2 of 3 stale cites and not at 1 of 3; a hidden stale document changes nothing; axe

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notice
- [x] UI and chat apps
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S26.1; tests green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
