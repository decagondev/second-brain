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
| Status | Done |

## Why

Stewards and security can't see where sensitive documents gather or whether that is changing.

## Acceptance criteria

- [x] Per space, weekly counts of documents with personal data, masked secrets, each label, wide sharing and holds
- [x] Shown on the review list (`section=sensitive`), in the admin's week (SB-5006) and at `GET /v1/admin/sensitive`
- [x] Counts only; a steward sees only their own spaces

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Weekly counts from the index
- [x] Review section and admin's week line
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. `Agent.SensitiveIn` counts a space's documents with personal data, a secret removed, each label, shared too widely and held, from the index; `Stewards.Sensitive` serves `review?section=sensitive` to its stewards only with last week's counts, `GET /v1/admin/sensitive` every space's to operators, and the admin's week gains a **Sensitive map** line per space with documents shared too widely or held. Each week's counts are kept in `state/sensitive/SPACE.json`. Counts only, never a title.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
