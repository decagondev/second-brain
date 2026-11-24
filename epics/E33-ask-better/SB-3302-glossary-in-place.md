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
| Status | Open |

## Why

Glossary terms are listed at the foot of the answer, not where they are used.

## Acceptance criteria

- [ ] Each term in `terms` is underlined where a claim uses it, as a whole word with the glossary's case rule
- [ ] Its meaning shows on hover and focus as plain text
- [ ] axe passes; a meaning with HTML shows as text

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] UI
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
