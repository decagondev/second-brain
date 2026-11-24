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
| Status | Open |

## Why

Who to ask names an owner, but not what else they own.

## Acceptance criteria

- [ ] `GET /v1/search` takes `owner=EMAIL`, which keeps only that owner's results after the live check
- [ ] Who to ask gets Their documents, which opens Documents with the filter
- [ ] An owner who isn't exactly one active user returns nothing

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Search
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
