---
title: "SB-2502: Claim a question"
summary: "Two stewards can answer the same question at once."
---

# SB-2502: Claim a question

| Field | Value |
|---|---|
| Epic | [E25 Easier to Steward](README.md) |
| Wave / slice | W24 / S24.1 |
| Stories | US-25.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-21 |
| Depends on | — |
| Status | Open |

## Why

Two stewards can answer the same question at once.

## Acceptance criteria

- [ ] I'll answer this shows who is answering to the other stewards for an hour, or until they reply or let go; it never blocks a reply
- [ ] Someone who is no longer a steward loses their claims at once
- [ ] A claim lapses after an hour (fake clock); a removed steward's claim disappears

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API and record
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
