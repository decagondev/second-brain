---
title: "SB-3003: Answers in the asker's language"
summary: "The UI speaks German and French, but answers come in whatever language the model writes."
---

# SB-3003: Answers in the asker's language

| Field | Value |
|---|---|
| Epic | [E30 Show the Evidence](README.md) |
| Wave / slice | W29 / S29.1 |
| Stories | US-30.1 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md), [`sb-evals(7)`](../../manual/sb-evals.7.md) |
| Laws | LAW-14, LAW-07 |
| Depends on | — |
| Status | Done |

## Why

The UI speaks German and French, but answers come in whatever language the model writes.

## Acceptance criteria

- [x] `POST /v1/ask` takes `language` (en, de or fr); the web UI sends the chosen one and chat apps send the manifest's
- [x] The prompt asks for claims in that language and keeps cites, titles and passages as they are
- [x] A German ask gets German claims with the same cites; an unknown code changes nothing; a German and a French eval case pass

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Ask
- [x] UI and chat apps
- [x] Eval cases
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S29.1; tested in internal/core/ask/evidence_test.go, internal/core/ingest/section_test.go, internal/adapters/connectors/folder/office_test.go (TestOfficeHeadings), web/src/evidence.test.tsx and the citation suite's cite-006 and cite-007.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
