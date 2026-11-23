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
| Status | Open |

## Why

Printing an answer prints the whole screen, and the cites' links are lost.

## Acceptance criteria

- [ ] A print stylesheet prints the question, the answer and its cites as numbered footnotes with their links and ages
- [ ] Save as PDF is the browser's print dialog; no server-side rendering
- [ ] A print-media snapshot test; the uncited mark prints

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Print stylesheet
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
