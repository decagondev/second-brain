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
| Status | Open |

## Why

Nobody checks that a restricted space's members still need it.

## Acceptance criteria

- [ ] Every 90 days a restricted space's stewards review its members: Still needs it or Remove, each audited
- [ ] Remove asks the admin to take the person out at the IdP; the cell changes no group
- [ ] Health warns 14 days overdue; an unread directory warns

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Notify job
- [ ] Steward screen and API
- [ ] Health
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
