---
title: "SB-2402: Print an answer"
summary: "Printing an answer prints the whole screen, and the cites' links are lost."
---

# SB-2402: Print an answer

| Field | Value |
|---|---|
| Epic | [E24 Easier to Ask](README.md) |
| Wave / slice | W23 / S23.1 |
| Stories | US-24.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-14 |
| Depends on | — |
| Status | Done |

## Why

Printing an answer prints the whole screen, and the cites' links are lost.

## Acceptance criteria

- [x] A print stylesheet prints the question, the answer and its cites as numbered footnotes with their links and ages
- [x] Save as PDF is the browser's print dialog; no server-side rendering
- [x] A print-media snapshot test; the uncited mark prints

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Print stylesheet
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Done 2026-11-23 (S23.1). **Print** opens the browser's dialog; the print stylesheet hides the tabs, the ask form and every button, hides the chips and prints the cites as numbered footnotes (`Footnotes` in `Answer.tsx`, the same numbers as the copy) with link and age; "Uncited" prints. jsdom can't apply print media, so the test checks the footnotes and the print rules in `styles.css` (read as text; `css: true` in the vitest config).

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
