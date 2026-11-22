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
| Status | Done |

## Why

Every question goes through the model, so a lookup is slow and costs tokens, and with the model down or the budget spent nothing answers.

## Acceptance criteria

- [x] `GET /v1/search?q=…[&space=…]` runs the ask path's retrieval, space checks and live ACL batch and returns up to 20 allowed documents, each once: title, `doc@version`, link, age and the best chunk's first 300 characters; no model call and no budget use
- [x] Every search gets a trace and an audit row like an ask; the query text is never logged, only its hash
- [x] The web UI has a Documents tab, and an ask that fails with `EUNAVAIL` or `EBUDGET` offers the same query as a search
- [x] Law tests: a document the asker can't open, or whose check is Unknown or late, is never returned (LAW-01, LAW-03)

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Endpoint on the retrieval path
- [x] Documents tab and the fallback offer
- [x] Tests, UI tests with axe
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

2026-11-22 (slice S16.1): `ask.Agent.Search` and `GET /v1/search` run the ask path's validation, retrieval, space checks and live ACL batch through `retrieve.ContextUpTo` (the same checks as `Context`, keeping every allowed chunk) and stop before the composer, so no model call and no budget. Up to 20 documents, each once, with title, cite, link, owner, `age_days` and the best chunk's first 300 characters. Chunks gained `Title` (the sanitised first line) and `Seen` (when the index first saw the version; re-ingesting the same version keeps the first time, in both index adapters). Each search stores a trace with `agent: search`, its decisions and the query's SHA-256, never its text; the weekly report counts searches apart. The web UI has Answer, Documents and Saved tabs, and an `EUNAVAIL` or `EBUDGET` ask offers "Search documents instead". Tests: `TestLAW01_SearchShowsOnlyWhatTheAskerCanOpen` (with its self-test, Unknown as Allow is caught), core and API tests, a running cell (`TestFindThroughARunningCell`, no tokens spent), and UI tests with axe.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
