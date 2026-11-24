---
title: "SB-3202: Refresh one document"
summary: "The only way to re-read one changed document is to wait for a sync."
---

# SB-3202: Refresh one document

| Field | Value |
|---|---|
| Epic | [E32 Run Without Surprises](README.md) |
| Wave / slice | W31 / S31.1 |
| Stories | US-32.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb(1)`](../../manual/sb.1.md) |
| Laws | LAW-11, LAW-13 |
| Depends on | — |
| Status | Done |

## Why

The only way to re-read one changed document is to wait for a sync.

## Acceptance criteria

- [x] `sb index refresh CITE --reason` and `POST /v1/admin/index/refresh` re-read one document through the normal pipeline, admins only, with an audit row
- [x] A document the source no longer has is tombstoned
- [x] A changed document has a new version after the refresh

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] CLI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

POST /v1/admin/index/refresh and sb index refresh CITE --reason re-read one document through the normal pipeline, audited before and after; a 404, 410 or missing file takes it out of the index, any other failure changes nothing

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
