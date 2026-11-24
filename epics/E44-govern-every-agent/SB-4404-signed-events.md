---
title: "SB-4404: Signed events out"
summary: "Other systems can't react to what happens in the brain without polling it."
---

# SB-4404: Signed events out

| Field | Value |
|---|---|
| Epic | [E44 Govern Every Agent](README.md) |
| Wave / slice | W43 / S43.2 |
| Stories | US-44.2 |
| Manual | [`sb-tenant(5)`](../../manual/sb-tenant.5.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-07, LAW-11 |
| Depends on | SB-2306 |
| Status | Open |

## Why

Other systems can't react to what happens in the brain without polling it.

## Acceptance criteria

- [ ] `events: { url, kinds }` is off by default; chosen kinds post the kind, space, id and time only
- [ ] Each body is signed with HMAC-SHA256 from a vault key reference, retried with backoff and audited
- [ ] A receiver fetches more with its own read-only token, checked as that token's person

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Manifest
- [ ] Delivery
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [18-legwork-inbox-govern.md](../../docs/18-legwork-inbox-govern.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
