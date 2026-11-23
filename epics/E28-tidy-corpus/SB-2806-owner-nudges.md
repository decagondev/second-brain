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
| Status | Open |

## Why

Owners never hear that answers rely on their old documents.

## Acceptance criteria

- [ ] On the first Monday of the month the notify job tells each active owner how many of their own often-cited documents are past review, as a new kind `owned` they can turn off
- [ ] Only documents the owner can open; counts only; a deprovisioned owner is told nothing
- [ ] Told once a month (fake clock)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Notify kind
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
