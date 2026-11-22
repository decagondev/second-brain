---
title: "SB-1701: Search without the model"
summary: "Every question goes through the model, so a lookup is slow and costs tokens, and with the model down or the budget spent nothing answers."
---

# SB-1701: Search without the model

| Field | Value |
|---|---|
| Epic | [E17 Team Knowledge](README.md) |
| Wave / slice | W16 / S16.1 |
| Stories | US-17.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-api(3)`](../../manual/sb-api.3.md) |
| Laws | LAW-01, LAW-03, LAW-14 |
| Depends on | — |
| Status | Proposed |

## Why

Every question goes through the model, so a lookup is slow and costs tokens, and with the model down or the budget spent nothing answers.

## Acceptance criteria

- [ ] `GET /v1/search?q=…[&space=…]` runs the ask path's retrieval, space checks and live ACL batch and returns up to 20 allowed documents, each once: title, `doc@version`, link, age and the best chunk's first 300 characters; no model call and no budget use
- [ ] Every search gets a trace and an audit row like an ask; the query text is never logged, only its hash
- [ ] The web UI has a Documents tab, and an ask that fails with `EUNAVAIL` or `EBUDGET` offers the same query as a search
- [ ] Law tests: a document the asker can't open, or whose check is Unknown or late, is never returned (LAW-01, LAW-03)

## Subtasks

- [ ] Update the man pages listed above first (manual first, P2)
- [ ] Endpoint on the retrieval path
- [ ] Documents tab and the fallback offer
- [ ] Tests, UI tests with axe
- [ ] Write or extend the law tests
- [ ] Devlog entry, ledger and changelog updated

## Progress

Proposed 2026-11-22 with W16 ([09-team-knowledge.md](../../docs/09-team-knowledge.md)). Not started.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
