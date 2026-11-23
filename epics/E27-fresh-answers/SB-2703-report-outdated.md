---
title: "SB-2703: Report as outdated"
summary: "A reader can report a wrong citation, but not an outdated one."
---

# SB-2703: Report as outdated

| Field | Value |
|---|---|
| Epic | [E27 Fresh Answers](README.md) |
| Wave / slice | W26 / S26.1 |
| Stories | US-27.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Done |

## Why

A reader can report a wrong citation, but not an outdated one.

## Acceptance criteria

- [x] `POST /v1/feedback` takes `outdated_citation`; the UI offers it next to wrong citation
- [x] It puts the document on the review list with `outdated`, a mark clears it, and it never becomes an eval case
- [x] A cite the reporter wasn't shown is refused

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] API
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S26.1; tests green.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
