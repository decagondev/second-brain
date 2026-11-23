---
title: "SB-2506: Pause notifications"
summary: "Someone on leave gets every message all the same."
---

# SB-2506: Pause notifications

| Field | Value |
|---|---|
| Epic | [E25 Easier to Steward](README.md) |
| Wave / slice | W24 / S24.2 |
| Stories | US-25.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | — |
| Depends on | — |
| Status | Open |

## Why

Someone on leave gets every message all the same.

## Acceptance criteria

- [ ] Pause for N days (1 to 30) in notification settings sends that person nothing, and the digest skips them; the pause ends by itself
- [ ] New questions still reach the space's other stewards
- [ ] A paused steward gets nothing; the other stewards are still told; the pause ends on its day (fake clock)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
