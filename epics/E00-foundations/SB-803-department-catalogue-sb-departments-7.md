---
title: "SB-803: Department catalogue: sb-departments(7)"
summary: "Companies organise by function; the catalogue says what each common department's brain holds and who reads it."
---

# SB-803: Department catalogue: `sb-departments(7)`

| Field | Value |
|---|---|
| Epic | [E00 Manual, Laws and Proofs](README.md) |
| Wave / slice | W8 / S8.1 Space pages |
| Stories | US-00.6 |
| Manual | `sb-departments(7)` (new), [`sb-onboarding(7)`](../../manual/sb-onboarding.7.md) |
| Laws | LAW-20 (new, W8) |
| Depends on | [SB-801](SB-801-sb-spaces-7-and-sb-space-5-first-drafts.md) |
| Status | Done |

## Why

Companies organise by function; the catalogue says what each common department's brain holds and who reads it.

## Acceptance criteria

- [x] Twelve templates plus `custom`, each with label default, sources, agents, retention and who else usually reads it
- [x] The four practices (roles and attributes, classification, need-to-know and barriers, joiner-mover-leaver with access reviews) explained in plain words
- [x] `sb-onboarding(7)` gains a 'choose your spaces' step

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Write the catalogue from docs/08-department-brains.md
- [x] Onboarding step
- [x] Glossary terms: space, steward, label, barrier
- [x] Write or extend the law tests: LAW-20 (red test in SB-804)
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-19 (slice S8.1): `sb-departments(7)` has the twelve templates plus `custom`, each with its label default, sources, agents, retention and who else usually reads it, the content that needs more than its default, and the four practices in plain words. sb-onboarding(7) Day 0 starts with a "choose your spaces" step, which has an expected result like every step and waits for SB-1001. The glossary already had space, steward, label and barrier from the plan. LAW-20's red test is SB-804's.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
