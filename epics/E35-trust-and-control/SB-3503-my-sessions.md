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
| Status | Open |

## Why

A person can sign out everywhere but can't see where they are signed in.

## Acceptance criteria

- [ ] `GET /v1/me/sessions` lists the caller's sessions and API tokens by kind, start, last used day and expiry, never an address or a token
- [ ] `DELETE /v1/me/sessions/{id}` ends one; the current one is marked
- [ ] Nobody else's session is listed or ended

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
