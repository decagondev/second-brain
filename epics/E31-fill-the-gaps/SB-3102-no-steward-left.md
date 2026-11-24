---
title: "SB-3102: Spaces with no steward left"
summary: "A space whose steward group emptied out still takes questions nobody gets."
---

# SB-3102: Spaces with no steward left

| Field | Value |
|---|---|
| Epic | [E31 Fill the Gaps](README.md) |
| Wave / slice | W30 / S30.1 |
| Stories | US-31.1 |
| Manual | [`sb-admin(3)`](../../manual/sb-admin.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-12, LAW-03 |
| Depends on | — |
| Status | Done |

## Why

A space whose steward group emptied out still takes questions nobody gets.

## Acceptance criteria

- [x] A space whose steward groups have no active member shows in health as `spaces_unstewarded` and warns
- [x] Its new questions go to the company-wide answerers, saying the space has no active steward
- [x] A directory error warns, never reads as fine

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Check
- [x] Routing
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

A question from a space with no active steward goes to company stewards with from_space; an unknown directory keeps it; health lists spaces_unstewarded

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
