---
title: "SB-2501: Reviewed, still right"
summary: "A stale document that is still right stays on the review list for good."
---

# SB-2501: Reviewed, still right

| Field | Value |
|---|---|
| Epic | [E25 Easier to Steward](README.md) |
| Wave / slice | W24 / S24.1 |
| Stories | US-25.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-spaces(7)`](../../manual/sb-spaces.7.md) |
| Laws | LAW-13, LAW-11 |
| Depends on | — |
| Status | Open |

## Why

A stale document that is still right stays on the review list for good.

## Acceptance criteria

- [ ] Reviewed marks a document at its current version, with an audit row; it leaves the list until a new version or `review_after_days` again
- [ ] A wrong-citation flag after the mark brings it back at once
- [ ] Mark, rebuild, still off; a new version, back on

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
