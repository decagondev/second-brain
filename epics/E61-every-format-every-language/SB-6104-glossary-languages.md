---
title: "SB-6104: A glossary in every language"
summary: "A glossary term has one word, so translations lose the company's own words."
---

# SB-6104: A glossary in every language

| Field | Value |
|---|---|
| Epic | [E61 Every Format, Every Language](README.md) |
| Wave / slice | W60 / S60.2 |
| Stories | US-61.2 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-16 |
| Depends on | SB-2203, SB-3302, SB-3107 |
| Status | Open |

## Why

A glossary term has one word, so translations lose the company's own words.

## Acceptance criteria

- [ ] A glossary term may carry its words per language (`in: {de: …, fr: …}`)
- [ ] Translated questions keep the glossary's word, and terms are marked in every language
- [ ] A translation is added through a proposal like any term change

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Glossary `in:` and its parser
- [ ] Marking terms in every language
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
