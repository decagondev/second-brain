---
title: "SB-3504: Access review"
summary: "Nobody checks that a restricted space's members still need it."
---

# SB-3504: Access review

| Field | Value |
|---|---|
| Epic | [E35 Trust and Control](README.md) |
| Wave / slice | W34 / S34.2 |
| Stories | US-35.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-admin(3)`](../../manual/sb-admin.3.md) |
| Laws | LAW-03, LAW-11, LAW-12 |
| Depends on | — |
| Status | Done |

## Why

Nobody checks that a restricted space's members still need it.

## Acceptance criteria

- [x] Every 90 days a restricted space's stewards review its members: Still needs it or Remove, each audited
- [x] Remove asks the admin to take the person out at the IdP; the cell changes no group
- [x] Health warns 14 days overdue; an unread directory warns

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notify job
- [x] Steward screen and API
- [x] Health
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S34.2 with tests; make check green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
