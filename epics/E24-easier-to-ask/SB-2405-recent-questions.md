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
| Status | Done |

## Why

People retype the question they asked an hour ago.

## Acceptance criteria

- [x] The ask box offers the person's last 20 questions, kept in their browser only and never sent to the cell for this
- [x] Cleared at sign-out and by Clear; a shared question isn't added
- [x] Sign-out clears them; a test confirms no request carries them

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Browser history
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-23 (S23.2). `web/src/recent.ts` keeps the last 20 questions in `localStorage` (`sb.recent`), each once, newest first; **Recent questions** under the ask box puts one back in the box, **Clear the list** empties it. Sign out everywhere and an `EAUTH` clear it; a shared question asked from its link isn't added. Tests (`findfaster.test.tsx`): no request made after a question carries an earlier one, sign-out and EAUTH clear the list, the cap, and storage that is off.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
