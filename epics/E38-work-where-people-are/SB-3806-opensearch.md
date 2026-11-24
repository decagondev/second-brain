---
title: "SB-3806: Search from the address bar"
summary: "Searching the brain means opening it first."
---

# SB-3806: Search from the address bar

| Field | Value |
|---|---|
| Epic | [E38 Work Where People Are](README.md) |
| Wave / slice | W37 / S37.2 |
| Stories | US-38.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | — |
| Depends on | SB-1701 |
| Status | Open |

## Why

Searching the brain means opening it first.

## Acceptance criteria

- [ ] The cell serves `/opensearch.xml` pointing at the Documents tab with the words as q
- [ ] The search runs signed in as always
- [ ] The file is valid OpenSearch

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Cell
- [ ] UI
- [ ] Tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
