---
title: "SB-3306: Their other documents"
summary: "Who to ask names an owner, but not what else they own."
---

# SB-3306: Their other documents

| Field | Value |
|---|---|
| Epic | [E33 Ask Better](README.md) |
| Wave / slice | W32 / S32.2 |
| Stories | US-33.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-12 |
| Depends on | SB-1702 |
| Status | Done |

## Why

Who to ask names an owner, but not what else they own.

## Acceptance criteria

- [x] `GET /v1/search` takes `owner=EMAIL`, which keeps only that owner's results after the live check
- [x] Who to ask gets Their documents, which opens Documents with the filter
- [x] An owner who isn't exactly one active user returns nothing

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Search
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

GET /v1/search takes owner=EMAIL (ask.SearchRequest.Owner): it keeps only that owner's results after every check, as shown, and with no q lists their documents from the index (Agent.ownerDocs, latest versions by title, retrieve.Allowed); Who to ask has Their documents, which opens Documents with the filter and Clear. Tests: TestTheirDocuments, web knowwhere.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
