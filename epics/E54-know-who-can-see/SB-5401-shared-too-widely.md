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
| Status | Done |

## Why

A sensitive document shared with the whole company is faithfully cited to the whole company, and nobody is told.

## Acceptance criteria

- [x] At each sync, a document open to the whole domain, `anyone` or a group above the cell's size limit that holds personal data, national ids, a masked secret, a sensitive label or sits in a restricted space is listed on the review list (`section=wide`) and the owner's inbox, with the widest grant and the reason
- [x] A wide document with nothing sensitive isn't listed
- [x] No text from the document appears on the list, and nothing is changed at the source

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Wide-grant check at sync
- [x] `section=wide` and the owner's inbox item
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. `ask.Wide` checks a document's ACL copy at read time: `anyone`, a `domain:` or a group above sb-cell(5) `wide_group_size` (default 500, sized from the directory), with `personal_data`, the new `secret_removed` flag (ingest keeps it on the document's other chunks when it drops one for a secret), a `confidential` or `restricted` label, or a restricted space. `Stewards.Wide` serves `GET /v1/spaces/NAME/review?section=wide` (grant, reasons, held, never the text), and `Agent.OwnWide` puts a `wide` item in the owner's inbox, linking to the source. Nothing at the source changes. Tested in TestKnowWhoCanSeeThroughARunningCell and TestWideSensitiveIsHeld.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
