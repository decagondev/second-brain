---
title: "SB-2805: Unused glossary terms"
summary: "Glossary terms pile up whether or not anyone uses them."
---

# SB-2805: Unused glossary terms

| Field | Value |
|---|---|
| Epic | [E28 A Tidy Corpus](README.md) |
| Wave / slice | W27 / S27.2 |
| Stories | US-28.2 |
| Manual | [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-07 |
| Depends on | — |
| Status | Done |

## Why

Glossary terms pile up whether or not anyone uses them.

## Acceptance criteria

- [x] The steward screen marks each glossary term no question in the space used in 180 days, from the traces' `terms`
- [x] Counts only; the manifest stays where terms are removed
- [x] A term used once in the window isn't marked

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Counts
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S27.2; tested in internal/core/ask/owned_test.go, internal/core/notify/owned_test.go, internal/core/report and web/src/tidy.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
