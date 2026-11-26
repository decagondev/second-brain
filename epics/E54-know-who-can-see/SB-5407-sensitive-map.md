---
title: "SB-5407: Sensitive map"
summary: "Stewards and security can't see where sensitive documents gather or whether that is changing."
---

# SB-5407: Sensitive map

| Field | Value |
|---|---|
| Epic | [E54 Know Who Can See](README.md) |
| Wave / slice | W53 / S53.2 |
| Stories | US-54.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-api(3)`](../../manual/sb-api.3.md), [`sb-observability(7)`](../../manual/sb-observability.7.md) |
| Laws | LAW-01, LAW-07 |
| Depends on | SB-5401, SB-5405 |
| Status | Open |

## Why

Stewards and security can't see where sensitive documents gather or whether that is changing.

## Acceptance criteria

- [ ] Per space, weekly counts of documents with personal data, masked secrets, each label, wide sharing and holds
- [ ] Shown on the review list (`section=sensitive`), in the admin's week (SB-5006) and at `GET /v1/admin/sensitive`
- [ ] Counts only; a steward sees only their own spaces

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Weekly counts from the index
- [ ] Review section and admin's week line
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
