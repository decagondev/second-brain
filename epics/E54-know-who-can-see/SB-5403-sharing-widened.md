---
title: "SB-5403: Sharing widened"
summary: "A document's sharing can widen between syncs without its owner noticing."
---

# SB-5403: Sharing widened

| Field | Value |
|---|---|
| Epic | [E54 Know Who Can See](README.md) |
| Wave / slice | W53 / S53.1 |
| Stories | US-54.1 |
| Manual | [`sb-ingest(7)`](../../manual/sb-ingest.7.md), [`sb-observability(7)`](../../manual/sb-observability.7.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-21 |
| Depends on | SB-5401, SB-2101 |
| Status | Done |

## Why

A document's sharing can widen between syncs without its owner noticing.

## Acceptance criteria

- [x] Each sync compares a document's ACL with the last one; a widening (a group, the domain, anyone with the link) sends its owner one notice naming the change
- [x] A sensitive document that widens also goes on the review list; a narrowing is recorded but not announced
- [x] When the owner can no longer open the document, the space's stewards are told instead

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] ACL diff between syncs
- [x] Notify kind `widened`
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-26: built. `ingest.Pipeline.Sharing` is told the grants added and removed when a sync's ACL copy differs from the indexed one (`ingest.SharingChangeOf`); the cell keeps each as an `ask.SharingEvent`. The notify job's new `Notices` hook sends a widening (a group, a domain or anyone added) once, kind `widened`, to the owner if they can still open the document at its source, else to its spaces' stewards. A narrowing is kept, not announced; a sensitive document that widened stays on `section=wide` for 30 days. An event that can't be kept counts `sb_sharing_unrecorded_total`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
