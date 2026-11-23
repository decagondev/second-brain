---
title: "SB-2405: Recent questions"
summary: "People retype the question they asked an hour ago."
---

# SB-2405: Recent questions

| Field | Value |
|---|---|
| Epic | [E24 Easier to Ask](README.md) |
| Wave / slice | W23 / S23.2 |
| Stories | US-24.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-07 |
| Depends on | — |
| Status | Open |

## Why

People retype the question they asked an hour ago.

## Acceptance criteria

- [ ] The ask box offers the person's last 20 questions, kept in their browser only and never sent to the cell for this
- [ ] Cleared at sign-out and by Clear; a shared question isn't added
- [ ] Sign-out clears them; a test confirms no request carries them

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Browser history
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
