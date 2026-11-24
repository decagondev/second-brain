---
title: "SB-3103: Review several at once"
summary: "Marks and retirements go one document at a time."
---

# SB-3103: Review several at once

| Field | Value |
|---|---|
| Epic | [E31 Fill the Gaps](README.md) |
| Wave / slice | W30 / S30.1 |
| Stories | US-31.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-11, LAW-19 |
| Depends on | SB-2501, SB-2801 |
| Status | Open |

## Why

Marks and retirements go one document at a time.

## Acceptance criteria

- [ ] `…/review/marks` and `…/retired` take `cites` (at most 50); each is checked and audited on its own, and failures are listed with why
- [ ] The review list has a checkbox each, Mark reviewed and Retire
- [ ] Marking 3 writes 3 audit rows; one outside the space fails alone

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] API
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
