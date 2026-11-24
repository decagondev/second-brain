---
title: "SB-3006: Open at the cited lines"
summary: "A GitHub cite opens the file at the top, not where the answer comes from."
---

# SB-3006: Open at the cited lines

| Field | Value |
|---|---|
| Epic | [E30 Show the Evidence](README.md) |
| Wave / slice | W29 / S29.2 |
| Stories | US-30.2 |
| Manual | [`sb-connector(3)`](../../manual/sb-connector.3.md) |
| Laws | — |
| Depends on | — |
| Status | Done |

## Why

A GitHub cite opens the file at the top, not where the answer comes from.

## Acceptance criteria

- [x] A GitHub cite's url opens the file on the default branch and ends with `#L12-L20` for the chunk's lines (the version is a commit time, so there is no commit to link)
- [x] Other kinds are unchanged
- [x] A span in lines 12 to 20 links to #L12-L20

## Subtasks

- [x] Update the man pages listed above first (manual first, P2)
- [x] Connector
- [x] Tests
- [x] Devlog entry, ledger and changelog updated

## Progress

Built in S29.2; tested in internal/core/ask/evidence_test.go (TestAskAboutDocuments, TestOneSourceWarning), tests/spaces/evidence_test.go, internal/adapters/connectors/github/github_test.go (TestLink) and web/src/evidence.test.tsx.

## Definition of done

See [docs/05-living-documents.md](../../docs/05-living-documents.md#definition-of-done).
