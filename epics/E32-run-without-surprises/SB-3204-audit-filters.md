---
title: "SB-3204: Audit by person and date"
summary: "The audit API filters by kind and space, not by who or when."
---

# SB-3204: Audit by person and date

| Field | Value |
|---|---|
| Epic | [E32 Run Without Surprises](README.md) |
| Wave / slice | W31 / S31.1 |
| Stories | US-32.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-audit(1)`](../../manual/sb-audit.1.md) |
| Laws | LAW-11 |
| Depends on | — |
| Status | Done |

## Why

The audit API filters by kind and space, not by who or when.

## Acceptance criteria

- [x] `GET /v1/admin/audit` takes `actor`, `from` and `to`; the console and `sb audit tail` get the same
- [x] Paging still works
- [x] The filters narrow as expected

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] Console and CLI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

GET /v1/admin/audit takes actor (a user id, or an email of one active user), from and to (days, inclusive); sb audit tail takes --since and --until, since --from was already the row number. The console has no audit list, so none was added

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
