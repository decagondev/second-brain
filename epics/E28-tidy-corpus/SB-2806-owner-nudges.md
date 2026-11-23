---
title: "SB-2806: Owners hear about their own old documents"
summary: "Owners never hear that answers rely on their old documents."
---

# SB-2806: Owners hear about their own old documents

| Field | Value |
|---|---|
| Epic | [E28 A Tidy Corpus](README.md) |
| Wave / slice | W27 / S27.2 |
| Stories | US-28.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01, LAW-07, LAW-12 |
| Depends on | — |
| Status | Done |

## Why

Owners never hear that answers rely on their old documents.

## Acceptance criteria

- [x] On the first Monday of the month the notify job tells each active owner how many of their own often-cited documents are past review, as a new kind `owned` they can turn off
- [x] Only documents the owner can open; counts only; a deprovisioned owner is told nothing
- [x] Told once a month (fake clock)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notify kind
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S27.2; tested in internal/core/ask/owned_test.go, internal/core/notify/owned_test.go, internal/core/report and web/src/tidy.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
