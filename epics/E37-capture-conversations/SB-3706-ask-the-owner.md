---
title: "SB-3706: Ask the owner"
summary: "Who to ask names the owner, but the asker has to find them elsewhere."
---

# SB-3706: Ask the owner

| Field | Value |
|---|---|
| Epic | [E37 Capture Conversations](README.md) |
| Wave / slice | W36 / S36.2 |
| Stories | US-37.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07, LAW-11, LAW-12 |
| Depends on | SB-3405 |
| Status | Done |

## Why

Who to ask names the owner, but the asker has to find them elsewhere.

## Acceptance criteria

- [x] Who to ask gets Ask them for an owner or caretaker who is one active user
- [x] It uses the steward question flow with that person as the only answerer; audited
- [x] A deprovisioned owner can't be asked

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Questions
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Ask them from Who to ask: the question goes to that one active person's own queue, replied to as a steward would

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
