---
title: "SB-3207: Notice to everyone"
summary: "There is no way to tell everyone the brain is down for an update."
---

# SB-3207: Notice to everyone

| Field | Value |
|---|---|
| Epic | [E32 Run Without Surprises](README.md) |
| Wave / slice | W31 / S31.2 |
| Stories | US-32.2 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-09, LAW-11 |
| Depends on | — |
| Status | Open |

## Why

There is no way to tell everyone the brain is down for an update.

## Acceptance criteria

- [ ] `PUT /v1/admin/notice` sets one plain line of at most 200 characters ending within 7 days; DELETE clears it; one record and an audit row
- [ ] The web UI shows it on every screen and chat replies put it first until it ends
- [ ] Too long or too far an end gets EINVAL; it ends by itself

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] UI and chat apps
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
