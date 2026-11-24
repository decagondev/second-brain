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
| Status | Open |

## Why

The only way to re-read one changed document is to wait for a sync.

## Acceptance criteria

- [ ] `sb index refresh CITE --reason` and `POST /v1/admin/index/refresh` re-read one document through the normal pipeline, admins only, with an audit row
- [ ] A document the source no longer has is tombstoned
- [ ] A changed document has a new version after the refresh

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] CLI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
