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
| Status | Done |

## Why

A glossary term has one word, so translations lose the company's own words.

## Acceptance criteria

- [x] A glossary term may carry its words per language (`in: {de: …, fr: …}`)
- [x] Translated questions keep the glossary's word, and terms are marked in every language
- [x] A translation is added through a proposal like any term change

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Glossary `in:` and its parser
- [x] Marking terms in every language
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-27: built. sb-space(5) `glossary_in` (TERM: {LANG: WORD}), checked against the glossary and inherited from the parent; `Term.Words` count as the term in a question and are marked in answers (web `glossed`); a proposal adds words to a term with `words`. The field is `glossary_in` beside `glossary`, not `in:` under each term, so `glossary` keeps its form. Tested in TestGlossaryIn and TestAskInOneLanguageFindAnother.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
