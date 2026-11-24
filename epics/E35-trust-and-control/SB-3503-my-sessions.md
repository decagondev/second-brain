---
title: "SB-3503: Where I'm signed in"
summary: "A person can sign out everywhere but can't see where they are signed in."
---

# SB-3503: Where I'm signed in

| Field | Value |
|---|---|
| Epic | [E35 Trust and Control](README.md) |
| Wave / slice | W34 / S34.1 |
| Stories | US-35.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-identity(7)`](../../manual/sb-identity.7.md) |
| Laws | LAW-11, LAW-12 |
| Depends on | SB-1906 |
| Status | Done |

## Why

A person can sign out everywhere but can't see where they are signed in.

## Acceptance criteria

- [x] `GET /v1/me/sessions` lists the caller's sessions and API tokens by kind, start, last used day and expiry, never an address or a token
- [x] `DELETE /v1/me/sessions/{id}` ends one; the current one is marked
- [x] Nobody else's session is listed or ended

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-24. `GET /v1/me/sessions` lists the caller's live sessions and tokens by kind, start, last-used day and end, with a public id (`ses_` and 16 hex of the key's SHA-256), never the key or a token, and `current` on this browser's. `DELETE /v1/me/sessions/{id}` ends one, audited first; any other id is `EPERM`. **Where I'm signed in** at the foot of the page. Tests: `TestOwnDataThroughARunningCell`, web `owndata.test.tsx`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
