---
title: "SB-3406: Your documents helped"
summary: "Owners hear about old documents but never that their documents helped."
---

# SB-3406: Your documents helped

| Field | Value |
|---|---|
| Epic | [E34 Capture Knowledge](README.md) |
| Wave / slice | W33 / S33.2 |
| Stories | US-34.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07 |
| Depends on | SB-2806 |
| Status | Done |

## Why

Owners hear about old documents but never that their documents helped.

## Acceptance criteria

- [x] The monthly owner note says how many helpful answers cited their documents last month
- [x] It names no asker or question and is left out at 0
- [x] Unhelpful answers don't count

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notify job
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-24. The owners' monthly note adds "Your documents were cited in N answers people found helpful last month": last calendar month's answers whose latest feedback is `up` and that cite a document the person owns (or cares for) and can open now. Counts only, left out at 0; someone with only this line gets it alone. Tests: `TestHelpedOwners`, `TestOwnersHearTheyHelped`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
