---
title: "SB-2201: Orphaned documents"
summary: "When someone leaves, their documents keep answering with no owner to ask."
---

# SB-2201: Orphaned documents

| Field | Value |
|---|---|
| Epic | [E22 Keep Knowledge Owned](README.md) |
| Wave / slice | W21 / S21.1 |
| Stories | US-22.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-12, LAW-01 |
| Depends on | — |
| Status | Done |

## Why

When someone leaves, their documents keep answering with no owner to ask.

## Acceptance criteria

- [x] The stewards' review list gains documents whose source owner is no active directory user, each checked live as the steward and shown as "owner gone"; those the steward can't open are counted
- [x] A deprovisioned owner is never shown as someone to ask
- [x] Deprovision an owner, and their document moves to the list within the next sync

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Owner check
- [x] Review list kind
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-23: built in S21.1. Owner check against the directory at read time (`Stewards.Gone`), `owner_gone` on items and hidden entries, at most 50. Law tests: `TestLAW12_ADeprovisionedOwnersDocumentsAreOnTheReviewList`, `TestLAW01_OwnerGoneNamesOnlyWhatTheStewardCanOpen`.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
