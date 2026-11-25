---
title: "SB-4802: A new joiner's first 90 days"
summary: "A new joiner's first weeks depend on a manager remembering what to send and when."
---

# SB-4802: A new joiner's first 90 days

| Field | Value |
|---|---|
| Epic | [E48 Ways of Working](README.md) |
| Wave / slice | W47 / S47.1 |
| Stories | US-48.1 |
| Manual | [`sb-onboarding(7)`](../../manual/sb-onboarding.7.md), [`sb-tenant(5)`](../../manual/sb-tenant.5.md) |
| Laws | LAW-01, LAW-12, LAW-29 |
| Depends on | SB-4801, SB-1706, SB-4506 |
| Status | Done |

## Why

A new joiner's first weeks depend on a manager remembering what to send and when.

## Acceptance criteria

- [x] The `onboarding` rhythm starts on SCIM add or by a steward, and every item is one the joiner can open
- [x] Unanswered questions reach stewards as gaps without the joiner's name unless they agree
- [x] `onboarding: false` turns it off for a space; leaving the space stops it

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Onboarding rhythm
- [x] People to meet
- [x] Gap hand-off
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-25: built. The built-in `onboarding` rhythm: the jobs service keeps a roster per space and starts it for each member SCIM added since its last run (the first run only notes who is there); the plan goes to the SCIM manager, or the stewards. Day 0 `start-here` lists the start-here items the joiner can open and the owners of them as people to meet; weeks 2 and 6 a catch-up since the start; the `gaps` steps tell the stewards how many of the joiner's asks found nothing and the glossary terms, named only after `POST /v1/rhythms/ID/consent`. `onboarding: false` removes it. TestRhythmsAndProjectsThroughARunningCell covers the start on SCIM add, the plan approval, the LAW-01 filtering and the stop on leaving.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
