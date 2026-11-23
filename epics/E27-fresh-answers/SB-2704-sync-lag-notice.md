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
| Status | Open |

## Why

An answer is silent when its connector stopped syncing.

## Acceptance criteria

- [ ] When a cite's connector is past its freshness target, the answer carries `notice_sync` with the kind and last success, and the UI says so
- [ ] An unreadable sync record is shown as unknown, never as fresh
- [ ] A fake clock past the target shows the notice

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Notice
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
