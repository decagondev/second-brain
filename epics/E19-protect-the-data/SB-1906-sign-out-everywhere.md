---
title: "SB-1906: Sign out everywhere"
summary: "A person who lost a laptop can't end their other sessions, and an operator can only deprovision them."
---

# SB-1906: Sign out everywhere

| Field | Value |
|---|---|
| Epic | [E19 Protect the Data](README.md) |
| Wave / slice | W18 / S18.2 |
| Stories | US-19.2 |
| Manual | [`sb-identity(7)`](../../manual/sb-identity.7.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-04, LAW-12 |
| Depends on | — |
| Status | Done |

## Why

A person who lost a laptop can't end their other sessions, and an operator can only deprovision them.

## Acceptance criteria

- [x] `POST /v1/sessions/revoke` ends every session of the caller
- [x] `POST /v1/admin/sessions/revoke` ends every session of a named person, policy-checked and audited before it runs
- [x] A revoked session is refused at its next request

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Sessions store
- [x] Endpoints and tests
- [x] UI link
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23 (slice S18.2): `session.Revoker` writes a `kind: access` audit row (`sessions:USER`, `revoke_requested`) before ending any session and a `revoked` row with the count after; if the first row fails nothing is revoked. `POST /v1/sessions/revoke` ends every session of the caller and clears the cookie; `POST /v1/admin/sessions/revoke` does it for a named person behind the operator policy, with a reason. Every request reads the session from the store, so a revoked one is `EAUTH` at once. The web UI has **Sign out everywhere** at its foot. `TestLAW12_SignOutEverywhereEndsEverySession` drives the real middleware, with self-tests for a revoker that ends one session and one that revokes before auditing.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
