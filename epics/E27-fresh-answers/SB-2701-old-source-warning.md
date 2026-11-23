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
| Status | Open |

## Why

An answer that rests mostly on old documents looks just like a fresh one.

## Acceptance criteria

- [ ] When more than half of an answer's cites are stale, the response carries `notice_stale` and the UI, Slack and Mattermost show one warning line first
- [ ] Only cites the asker can see are counted
- [ ] The warning shows at 2 of 3 stale cites and not at 1 of 3; a hidden stale document changes nothing; axe

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Notice
- [ ] UI and chat apps
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
