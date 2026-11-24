---
title: "SB-3803: Every token in one place"
summary: "An admin can't see which API tokens exist across the tenant."
---

# SB-3803: Every token in one place

| Field | Value |
|---|---|
| Epic | [E38 Work Where People Are](README.md) |
| Wave / slice | W37 / S37.1 |
| Stories | US-38.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-11, LAW-12 |
| Depends on | SB-2306 |
| Status | Open |

## Why

An admin can't see which API tokens exist across the tenant.

## Acceptance criteria

- [ ] `GET /v1/admin/tokens` lists every live token: owner, name, created, last used day, expiry, never the token
- [ ] `DELETE /v1/admin/tokens/{id}` revokes one at once, audited with a reason
- [ ] The admin console lists them, oldest unused first

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] Admin console
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
