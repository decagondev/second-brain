---
title: "SB-2503: Reply templates"
summary: "Stewards type the same replies again and again."
---

# SB-2503: Reply templates

| Field | Value |
|---|---|
| Epic | [E25 Easier to Steward](README.md) |
| Wave / slice | W24 / S24.1 |
| Stories | US-25.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-19 |
| Depends on | — |
| Status | Done |

## Why

Stewards type the same replies again and again.

## Acceptance criteria

- [x] Each space keeps up to 20 replies (a name and up to 2,000 characters) its stewards write and edit; Use a template fills the editable reply box
- [x] Templates are never sent to the model
- [x] A steward of another space never sees them

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API and record
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S24.1; see manual/sb-ask.3.md

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
