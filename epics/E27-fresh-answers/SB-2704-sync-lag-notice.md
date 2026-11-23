---
title: "SB-2704: Sync lag notice"
summary: "An answer is silent when its connector stopped syncing."
---

# SB-2704: Sync lag notice

| Field | Value |
|---|---|
| Epic | [E27 Fresh Answers](README.md) |
| Wave / slice | W26 / S26.1 |
| Stories | US-27.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-03 |
| Depends on | — |
| Status | Done |

## Why

An answer is silent when its connector stopped syncing.

## Acceptance criteria

- [x] When a cite's connector is past its freshness target, the answer carries `notice_sync` with the kind and last success, and the UI says so
- [x] An unreadable sync record is shown as unknown, never as fresh
- [x] A fake clock past the target shows the notice

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notice
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S26.1; tests green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
