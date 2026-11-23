---
title: "SB-2203: Space glossary"
summary: "Search misses documents that spell out the acronym the asker used."
---

# SB-2203: Space glossary

| Field | Value |
|---|---|
| Epic | [E22 Keep Knowledge Owned](README.md) |
| Wave / slice | W21 / S21.2 |
| Stories | US-22.2 |
| Manual | [`sb-space(5)`](../../manual/sb-space.5.md), [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-09, LAW-19 |
| Depends on | — |
| Status | Open |

## Why

Search misses documents that spell out the acronym the asker used.

## Acceptance criteria

- [ ] A space's stewards keep up to 200 terms (`term`, `means`), proposed like any space change
- [ ] Search expands a query's terms with their meanings in that space only; an answer that uses a term shows it at the foot
- [ ] Terms go in the data block, never as instructions; an entry that reads like an instruction is flagged and not used

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Glossary in the space file
- [ ] Query expansion
- [ ] Answer foot
- [ ] Tests
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

—

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
