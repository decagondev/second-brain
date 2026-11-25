---
title: "SB-4706: Marked as AI"
summary: "A published note or message can't be told apart from one a person wrote."
---

# SB-4706: Marked as AI

| Field | Value |
|---|---|
| Epic | [E47 Every Agent Accountable](README.md) |
| Wave / slice | W46 / S46.2 |
| Stories | US-47.2 |
| Manual | [`sb-agents(7)`](../../manual/sb-agents.7.md), [`sb-laws(7)`](../../manual/sb-laws.7.md) |
| Laws | LAW-28, LAW-11 |
| Depends on | SB-4402 |
| Status | Open |

## Why

A published note or message can't be told apart from one a person wrote.

## Acceptance criteria

- [ ] Every draft, message, email and chat post an agent writes says which agent and version wrote it
- [ ] A published note keeps `generated_by` and `approved_by`, and its cite shows the mark; exports and prints keep it
- [ ] LAW-28 is a law test and goes green; a note a person rewrote loses the mark

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Mark
- [ ] Front matter
- [ ] UI and chat
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started. The concept is in [19-departments-shape-account.md](../../docs/19-departments-shape-account.md).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
