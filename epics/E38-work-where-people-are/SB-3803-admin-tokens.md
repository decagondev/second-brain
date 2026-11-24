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
| Status | Done |

## Why

An admin can't see which API tokens exist across the tenant.

## Acceptance criteria

- [x] `GET /v1/admin/tokens` lists every live token: owner, name, created, last used day, expiry, never the token
- [x] `DELETE /v1/admin/tokens/{id}` revokes one at once, audited with a reason
- [x] The admin console lists them, oldest unused first

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] Admin console
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-24: built. `internal/api/admintokens.go`: `GET /v1/admin/tokens` (`api.AdminTokens`, oldest unused first, never the token or its hash) and `DELETE /v1/admin/tokens/{id}` with a reason, audited `token_revoked` first; `Admin.Tokens` wired to the session store's `ListTokens`. Web `AdminTokens` in the admin console. TestAdminTokensOrder, the running-cell test, web reach.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
