---
title: "SB-2401: Copy an answer with its cites"
summary: "An answer can't be pasted into an email or a ticket with its cites."
---

# SB-2401: Copy an answer with its cites

| Field | Value |
|---|---|
| Epic | [E24 Easier to Ask](README.md) |
| Wave / slice | W23 / S23.1 |
| Stories | US-24.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-14, LAW-01 |
| Depends on | — |
| Status | Open |

## Why

An answer can't be pasted into an email or a ticket with its cites.

## Acceptance criteria

- [ ] A Copy button puts the answer on the clipboard as plain text and Markdown, each claim followed by its cite's title and link
- [ ] An uncited claim keeps the screen's "not cited" mark; hidden sources are only their count
- [ ] The copy matches the screen; a test fails if an uncited claim loses its mark

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Copy button and formats
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
