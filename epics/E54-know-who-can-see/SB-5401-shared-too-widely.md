---
title: "SB-5401: Shared too widely"
summary: "A sensitive document shared with the whole company is faithfully cited to the whole company, and nobody is told."
---

# SB-5401: Shared too widely

| Field | Value |
|---|---|
| Epic | [E54 Know Who Can See](README.md) |
| Wave / slice | W53 / S53.1 |
| Stories | US-54.1 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md), [`sb-permissions(7)`](../../manual/sb-permissions.7.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-07, LAW-19 |
| Depends on | SB-1906, SB-2207, SB-5102 |
| Status | Open |

## Why

A sensitive document shared with the whole company is faithfully cited to the whole company, and nobody is told.

## Acceptance criteria

- [ ] At each sync, a document open to the whole domain, `anyone` or a group above the cell's size limit that holds personal data, national ids, a masked secret, a sensitive label or sits in a restricted space is listed on the review list (`section=wide`) and the owner's inbox, with the widest grant and the reason
- [ ] A wide document with nothing sensitive isn't listed
- [ ] No text from the document appears on the list, and nothing is changed at the source

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Wide-grant check at sync
- [ ] `section=wide` and the owner's inbox item
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
