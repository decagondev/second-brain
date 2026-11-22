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
| Status | Open |

## Why

A person who lost a laptop can't end their other sessions, and an operator can only deprovision them.

## Acceptance criteria

- [ ] `POST /v1/sessions/revoke` ends every session of the caller
- [ ] `POST /v1/admin/sessions/revoke` ends every session of a named person, policy-checked and audited before it runs
- [ ] A revoked session is refused at its next request

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Sessions store
- [ ] Endpoints and tests
- [ ] UI link
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
