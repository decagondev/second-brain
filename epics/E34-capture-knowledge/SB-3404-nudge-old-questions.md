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
| Status | Done |

## Why

An open question to the stewards can wait forever.

## Acceptance criteria

- [x] The notify job marks a question open 5 working days overdue and lists it first in the stewards' digest
- [x] At 10 working days it goes to the company-wide answerers with how long it waited
- [x] A claimed question waits for its claim to lapse

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Notify job
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-24. Working days are Monday to Friday in the digest zone. An open question is `overdue` from 5 (marked on the steward screen, counted first in the digest line, those spaces first). From 10, with no live claim, the notify job shows it to the company-wide answerers too (`also`, `waited`), audited as `rerouted` by `sys_notify`, and tells them a count; it stays in its own queue, and a note of it is published from its own space only. Tests: `TestOldQuestionsWaitThenGoCompanyWide`, `TestDigestOverdueFirst`, `TestWorkingDays`, `TestOverdueAndWaitedOnTheQueue`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
