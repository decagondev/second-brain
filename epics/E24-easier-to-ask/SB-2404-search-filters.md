---
title: "SB-2404: Search filters"
summary: "Search can't be narrowed to a kind of file or to recent documents."
---

# SB-2404: Search filters

| Field | Value |
|---|---|
| Epic | [E24 Easier to Ask](README.md) |
| Wave / slice | W23 / S23.2 |
| Stories | US-24.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Open |

## Why

Search can't be narrowed to a kind of file or to recent documents.

## Acceptance criteria

- [ ] `GET /v1/search` takes `type=` (from the extension on the source id or title; none is `other`) and `since=Nd` (the version's first-seen time)
- [ ] The Documents tab shows both as drop-downs; filters run after the live ACL check
- [ ] Each filter narrows; a document the person can't open never appears, whatever the filters

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API filters
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
