---
title: "SB-3004: Ask about this document"
summary: "There is no way to ask about one particular document."
---

# SB-3004: Ask about this document

| Field | Value |
|---|---|
| Epic | [E30 Show the Evidence](README.md) |
| Wave / slice | W29 / S29.2 |
| Stories | US-30.2 |
| Manual | [`sb-ask(3)`](../../manual/sb-ask.3.md) |
| Laws | LAW-01 |
| Depends on | — |
| Status | Done |

## Why

There is no way to ask about one particular document.

## Acceptance criteria

- [x] A chip and a search result offer Ask about this document; the next ask carries `documents` (at most 5) and retrieval keeps only their chunks, after the checks
- [x] A chip above the ask box shows the filter, with Clear
- [x] An answer cites only the chosen document; a hidden one gives no_access and nothing else

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Ask
- [x] UI
- [x] Tests
- [x] Write or extend the law tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S29.2; tested in internal/core/ask/evidence_test.go (TestAskAboutDocuments, TestOneSourceWarning), tests/spaces/evidence_test.go, internal/adapters/connectors/github/github_test.go (TestLink) and web/src/evidence.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
