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
| Status | Done |

## Why

An answer can't be pasted into an email or a ticket with its cites.

## Acceptance criteria

- [x] A Copy button puts the answer on the clipboard as plain text and Markdown, each claim followed by its cite's title and link
- [x] An uncited claim keeps the screen's "not cited" mark; hidden sources are only their count
- [x] The copy matches the screen; a test fails if an uncited claim loses its mark

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Copy button and formats
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-23 (S23.1). **Copy** and **Copy as Markdown** under each answer (`web/src/Copy.tsx`, `answerText`, `answerMarkdown`): the question, each claim with its cite numbers, an uncited one as "Uncited: …", the numbered cites with title and link, and the screen's hidden-sources line; never the trace, an owner or anything else. Markdown escapes the claim text and links only http(s) cites. Tests: `web/src/copy.test.tsx` (exact text, the uncited mark in both forms, escaping, the button).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
