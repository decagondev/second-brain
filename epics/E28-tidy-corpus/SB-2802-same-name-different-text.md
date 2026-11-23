---
title: "SB-2802: Same name, different text"
summary: "Two documents with the same name and different text both answer, and nobody notices."
---

# SB-2802: Same name, different text

| Field | Value |
|---|---|
| Epic | [E28 A Tidy Corpus](README.md) |
| Wave / slice | W27 / S27.1 |
| Stories | US-28.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Open |

## Why

Two documents with the same name and different text both answer, and nobody notices.

## Acceptance criteria

- [ ] The review list groups documents the space holds with the same title (case and spacing ignored) and different text, newest first
- [ ] Documents the steward can't open are counts with their owner
- [ ] Two different Leave policy files are grouped; a hidden one is a count only

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Review section
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
