---
title: "SB-3302: Glossary where it's used"
summary: "Glossary terms are listed at the foot of the answer, not where they are used."
---

# SB-3302: Glossary where it's used

| Field | Value |
|---|---|
| Epic | [E33 Ask Better](README.md) |
| Wave / slice | W32 / S32.1 |
| Stories | US-33.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-09, LAW-14 |
| Depends on | SB-2203 |
| Status | Done |

## Why

Glossary terms are listed at the foot of the answer, not where they are used.

## Acceptance criteria

- [x] Each term in `terms` is underlined where a claim uses it, as a whole word with the glossary's case rule
- [x] Its meaning shows on hover and focus as plain text
- [x] axe passes; a meaning with HTML shows as text

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

The UI's glossed() underlines each of the answer's terms where a claim uses it, as a whole word with the glossary's case rule, and TermWord shows the meaning as plain text on hover and focus (role tooltip, aria-describedby). Test: web askbetter.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
