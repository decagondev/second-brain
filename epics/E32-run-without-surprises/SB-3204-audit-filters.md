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
| Status | Open |

## Why

The audit API filters by kind and space, not by who or when.

## Acceptance criteria

- [ ] `GET /v1/admin/audit` takes `actor`, `from` and `to`; the console and `sb audit tail` get the same
- [ ] Paging still works
- [ ] The filters narrow as expected

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] Console and CLI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
