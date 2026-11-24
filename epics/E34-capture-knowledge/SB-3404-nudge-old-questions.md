---
title: "SB-3404: Nudge old questions"
summary: "An open question to the stewards can wait forever."
---

# SB-3404: Nudge old questions

| Field | Value |
|---|---|
| Epic | [E34 Capture Knowledge](README.md) |
| Wave / slice | W33 / S33.1 |
| Stories | US-34.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07 |
| Depends on | SB-2502 |
| Status | Open |

## Why

An open question to the stewards can wait forever.

## Acceptance criteria

- [ ] The notify job marks a question open 5 working days overdue and lists it first in the stewards' digest
- [ ] At 10 working days it goes to the company-wide answerers with how long it waited
- [ ] A claimed question waits for its claim to lapse

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Notify job
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
