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
| Status | Open |

## Why

A document's sharing can widen between syncs without its owner noticing.

## Acceptance criteria

- [ ] Each sync compares a document's ACL with the last one; a widening (a group, the domain, anyone with the link) sends its owner one notice naming the change
- [ ] A sensitive document that widens also goes on the review list; a narrowing is recorded but not announced
- [ ] When the owner can no longer open the document, the space's stewards are told instead

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] ACL diff between syncs
- [ ] Notify kind `widened`
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
